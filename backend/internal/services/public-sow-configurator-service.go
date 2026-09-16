package services

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/pixels-two/sow/backend/internal/common"
	"github.com/pixels-two/sow/backend/internal/mail"
	"github.com/pixels-two/sow/backend/internal/models/dto"
)

// PublicSOWConfiguratorService validates and delivers public configured agreements.
type PublicSOWConfiguratorService struct {
	generator PublicSOWDocumentGenerator
	mailer    mail.Mailer
	mu        sync.Mutex
	pending   map[string]*pendingDelivery
}

// NewPublicSOWConfiguratorService builds the public configurator service.
func NewPublicSOWConfiguratorService(generator PublicSOWDocumentGenerator, mailer mail.Mailer) *PublicSOWConfiguratorService {
	return &PublicSOWConfiguratorService{generator: generator, mailer: mailer, pending: make(map[string]*pendingDelivery)}
}

// Download validates a public request and returns its configured document.
func (s *PublicSOWConfiguratorService) Download(_ context.Context, request dto.PublicSOWDownloadRequest) (DocumentArtifact, error) {
	if !bool(request.Acknowledged) {
		return DocumentArtifact{}, acknowledgmentError()
	}
	configuration, err := validateAndNormalizePublicSOW(request.Answers, request.ClientInformation, "download")
	if err != nil {
		return DocumentArtifact{}, err
	}
	artifact, err := s.generator.Generate(configuration)
	if err != nil {
		return DocumentArtifact{}, fmt.Errorf("generating public SOW document: %w", common.ErrGenerationFailed)
	}
	return artifact, nil
}

// prepareEmailDelivery validates a public email request and the mail
// provider's availability ahead of digesting and sending.
func (s *PublicSOWConfiguratorService) prepareEmailDelivery(request dto.PublicSOWEmailRequest, key string) (string, PublicSOWConfiguration, error) {
	if !bool(request.Acknowledged) {
		return "", PublicSOWConfiguration{}, acknowledgmentError()
	}
	recipient, err := normalizeEmail(string(request.Email))
	if err != nil {
		return "", PublicSOWConfiguration{}, err
	}
	if err := validateIdempotencyKey(key); err != nil {
		return "", PublicSOWConfiguration{}, err
	}
	configuration, err := validateAndNormalizePublicSOW(request.Answers, request.ClientInformation, "email")
	if err != nil {
		return "", PublicSOWConfiguration{}, err
	}
	reporter, ok := s.mailer.(mail.AvailabilityReporter)
	if !ok || !reporter.Available() {
		return "", PublicSOWConfiguration{}, fmt.Errorf("checking public SOW mail provider: %w", common.ErrEmailDeliveryUnavailable)
	}
	return recipient, configuration, nil
}

// Email validates a public request and sends its configured Word attachment.
func (s *PublicSOWConfiguratorService) Email(ctx context.Context, request dto.PublicSOWEmailRequest, key string) (DocumentArtifact, error) {
	recipient, configuration, err := s.prepareEmailDelivery(request, key)
	if err != nil {
		return DocumentArtifact{}, err
	}
	digest, err := publicSOWDeliveryDigest(recipient, configuration)
	if err != nil {
		return DocumentArtifact{}, fmt.Errorf("hashing public SOW delivery: %w", common.ErrGenerationFailed)
	}
	entry, leader, err := s.begin(key, digest)
	if err != nil {
		return DocumentArtifact{}, err
	}
	if !leader {
		select {
		case <-ctx.Done():
			return DocumentArtifact{}, fmt.Errorf("waiting for duplicate public SOW delivery: %w", ctx.Err())
		case <-entry.done:
			return DocumentArtifact{}, entry.err
		}
	}

	artifact, sendErr := s.generateAndSend(ctx, recipient, key, configuration)
	s.finish(key, entry, sendErr)
	return artifact, sendErr
}

func (s *PublicSOWConfiguratorService) generateAndSend(ctx context.Context, recipient, key string, configuration PublicSOWConfiguration) (DocumentArtifact, error) {
	artifact, err := s.generator.Generate(configuration)
	if err != nil {
		return DocumentArtifact{}, fmt.Errorf("generating public SOW email attachment: %w", common.ErrGenerationFailed)
	}
	message := mail.Message{
		To:             recipient,
		Subject:        "Your configured content development agreement",
		Text:           "Your requested editable Word agreement is attached. Review and adapt it with qualified legal counsel before use.",
		IdempotencyKey: key,
		Attachments:    []mail.Attachment{{Filename: artifact.Filename, ContentType: artifact.MIMEType, Content: artifact.Bytes}},
	}
	if err := s.mailer.Send(ctx, message); err != nil {
		return DocumentArtifact{}, fmt.Errorf("sending public SOW email: %v: %w", err, common.ErrEmailDeliveryUnavailable)
	}
	return artifact, nil
}

func (s *PublicSOWConfiguratorService) begin(key string, digest [32]byte) (*pendingDelivery, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if existing := s.pending[key]; existing != nil {
		if existing.digest != digest {
			return nil, false, fmt.Errorf("checking public SOW delivery key: %w", common.ErrIdempotencyConflict)
		}
		return existing, false, nil
	}
	entry := &pendingDelivery{digest: digest, done: make(chan struct{})}
	s.pending[key] = entry
	return entry, true, nil
}

func (s *PublicSOWConfiguratorService) finish(key string, entry *pendingDelivery, err error) {
	s.mu.Lock()
	entry.err = err
	delete(s.pending, key)
	close(entry.done)
	s.mu.Unlock()
}

func publicSOWDeliveryDigest(recipient string, configuration PublicSOWConfiguration) ([32]byte, error) {
	body, err := json.Marshal(struct {
		Recipient     string                 `json:"recipient"`
		Configuration PublicSOWConfiguration `json:"configuration"`
	}{recipient, configuration})
	if err != nil {
		return [32]byte{}, fmt.Errorf("encoding public SOW delivery digest: %w", err)
	}
	return sha256.Sum256(body), nil
}
