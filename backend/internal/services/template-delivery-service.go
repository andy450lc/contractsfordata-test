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

// TemplateDeliveryService creates request-scoped documents and delivers them.
type TemplateDeliveryService struct {
	generator DocumentGenerator
	mailer    mail.Mailer
	mu        sync.Mutex
	pending   map[string]*pendingDelivery
}

type pendingDelivery struct {
	digest [32]byte
	done   chan struct{}
	err    error
}

// NewTemplateDeliveryService builds the shared download and email service.
func NewTemplateDeliveryService(generator DocumentGenerator, mailer mail.Mailer) *TemplateDeliveryService {
	return &TemplateDeliveryService{generator: generator, mailer: mailer, pending: make(map[string]*pendingDelivery)}
}

// Download validates acknowledgment and returns one generated document.
func (s *TemplateDeliveryService) Download(ctx context.Context, acknowledged bool, configuration *dto.ContractConfiguration) (DocumentArtifact, error) {
	if !acknowledged {
		return DocumentArtifact{}, acknowledgmentError()
	}
	if err := validateConfiguration(configuration); err != nil {
		return DocumentArtifact{}, err
	}
	artifact, err := s.generator.Generate(configuration)
	if err != nil {
		return DocumentArtifact{}, fmt.Errorf("generating template document: %w", common.ErrGenerationFailed)
	}
	return artifact, nil
}

// prepareEmailDelivery validates an authenticated email request and the
// mail provider's availability ahead of digesting and sending.
func (s *TemplateDeliveryService) prepareEmailDelivery(acknowledged bool, recipient, key string, configuration *dto.ContractConfiguration) (string, error) {
	if !acknowledged {
		return "", acknowledgmentError()
	}
	normalized, err := normalizeEmail(recipient)
	if err != nil {
		return "", err
	}
	if err := validateIdempotencyKey(key); err != nil {
		return "", err
	}
	if err := validateConfiguration(configuration); err != nil {
		return "", err
	}
	reporter, ok := s.mailer.(mail.AvailabilityReporter)
	if !ok || !reporter.Available() {
		return "", fmt.Errorf("checking mail provider: %w", common.ErrEmailDeliveryUnavailable)
	}
	return normalized, nil
}

// Email validates the request, generates one document, and sends its exact bytes.
func (s *TemplateDeliveryService) Email(ctx context.Context, acknowledged bool, recipient string, key string, configuration *dto.ContractConfiguration) (DocumentArtifact, error) {
	normalized, err := s.prepareEmailDelivery(acknowledged, recipient, key, configuration)
	if err != nil {
		return DocumentArtifact{}, err
	}

	digest, err := deliveryDigest(normalized, configuration)
	if err != nil {
		return DocumentArtifact{}, fmt.Errorf("hashing delivery request: %w", common.ErrGenerationFailed)
	}
	entry, leader, err := s.begin(key, digest)
	if err != nil {
		return DocumentArtifact{}, err
	}
	if !leader {
		select {
		case <-ctx.Done():
			return DocumentArtifact{}, fmt.Errorf("waiting for duplicate delivery: %w", ctx.Err())
		case <-entry.done:
			return DocumentArtifact{}, entry.err
		}
	}

	artifact, sendErr := s.generateAndSend(ctx, normalized, key, configuration)
	s.finish(key, entry, sendErr)
	return artifact, sendErr
}

func (s *TemplateDeliveryService) generateAndSend(ctx context.Context, recipient, key string, configuration *dto.ContractConfiguration) (DocumentArtifact, error) {
	artifact, err := s.generator.Generate(configuration)
	if err != nil {
		return DocumentArtifact{}, fmt.Errorf("generating email attachment: %w", common.ErrGenerationFailed)
	}
	message := mail.Message{To: recipient, Subject: "Your editable content development agreement", Text: "Your editable Content Development Agreement is attached. Review and adapt it with qualified legal counsel before use.", IdempotencyKey: key, Attachments: []mail.Attachment{{Filename: artifact.Filename, ContentType: artifact.MIMEType, Content: artifact.Bytes}}}
	if err := s.mailer.Send(ctx, message); err != nil {
		return DocumentArtifact{}, fmt.Errorf("sending template email: %v: %w", err, common.ErrEmailDeliveryUnavailable)
	}
	return artifact, nil
}

func (s *TemplateDeliveryService) begin(key string, digest [32]byte) (*pendingDelivery, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if existing := s.pending[key]; existing != nil {
		if existing.digest != digest {
			return nil, false, fmt.Errorf("checking delivery key: %w", common.ErrIdempotencyConflict)
		}
		return existing, false, nil
	}
	entry := &pendingDelivery{digest: digest, done: make(chan struct{})}
	s.pending[key] = entry
	return entry, true, nil
}

func (s *TemplateDeliveryService) finish(key string, entry *pendingDelivery, err error) {
	s.mu.Lock()
	entry.err = err
	delete(s.pending, key)
	close(entry.done)
	s.mu.Unlock()
}

func deliveryDigest(recipient string, configuration *dto.ContractConfiguration) ([32]byte, error) {
	body, err := json.Marshal(struct {
		Recipient     string                     `json:"recipient"`
		Configuration *dto.ContractConfiguration `json:"configuration"`
	}{recipient, configuration})
	if err != nil {
		return [32]byte{}, fmt.Errorf("encoding delivery digest: %w", err)
	}
	return sha256.Sum256(body), nil
}

func acknowledgmentError() error {
	return &common.ValidationError{Details: []common.ValidationDetail{{Field: "acknowledged", Message: "must be true"}}}
}
