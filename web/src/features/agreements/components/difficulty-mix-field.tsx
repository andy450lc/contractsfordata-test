import { Slider as SliderPrimitive } from 'radix-ui'

import { cn } from '@/lib/utils'

import { FieldError, NumberInput } from './field'

export interface DifficultyMix {
  easy: number
  medium: number
  hard: number
}

const levels = ['easy', 'medium', 'hard'] as const
type Level = (typeof levels)[number]

const labels: Record<Level, string> = { easy: 'Easy', medium: 'Medium', hard: 'Hard' }

const colors: Record<Level, string> = {
  easy: 'bg-foreground/25',
  medium: 'bg-foreground/55',
  hard: 'bg-foreground',
}

function clamp(value: number): number {
  if (!Number.isFinite(value)) return 0
  return Math.min(100, Math.max(0, Math.round(value)))
}

// rebalance applies one edited share and adjusts the others so the three
// still add up to 100. The neighbouring share absorbs the change. The
// far share moves only when the neighbour reaches zero.
export function rebalance(
  mix: DifficultyMix,
  level: Level,
  value: number,
): DifficultyMix {
  const next = { ...mix, [level]: clamp(value) }
  const order: Record<Level, [Level, Level]> = {
    easy: ['medium', 'hard'],
    medium: ['hard', 'easy'],
    hard: ['medium', 'easy'],
  }
  const [neighbour, far] = order[level]
  const remainder = 100 - next[level]
  next[far] = Math.min(next[far], remainder)
  next[neighbour] = remainder - next[far]
  return next
}

interface DifficultyMixFieldProps {
  value: DifficultyMix
  onChange: (value: DifficultyMix) => void
  error?: string
}

// DifficultyMixField edits three shares that sum to 100 with one track
// and two thumbs. The number fields below stay in sync with the track.
export function DifficultyMixField({ value, onChange, error }: DifficultyMixFieldProps) {
  const easy = clamp(value.easy)
  const medium = clamp(value.medium)
  const hard = clamp(value.hard)
  const stops = [easy, easy + medium]

  function fromStops([firstStop = 0, secondStop = 100]: number[]) {
    const nextEasy = clamp(firstStop)
    const nextMedium = clamp(secondStop) - nextEasy
    onChange({ easy: nextEasy, medium: nextMedium, hard: 100 - nextEasy - nextMedium })
  }

  return (
    <div className="flex flex-col gap-4">
      <div className="flex flex-col gap-2">
        <SliderPrimitive.Root
          value={stops}
          onValueChange={fromStops}
          min={0}
          max={100}
          step={1}
          className="relative flex h-5 w-full touch-none items-center select-none"
        >
          <SliderPrimitive.Track className="relative h-2 w-full overflow-hidden rounded-full bg-muted">
            <span
              className={cn('absolute inset-y-0 left-0', colors.easy)}
              style={{ width: `${easy}%` }}
            />
            <span
              className={cn('absolute inset-y-0', colors.medium)}
              style={{ left: `${easy}%`, width: `${medium}%` }}
            />
            <span
              className={cn('absolute inset-y-0 right-0', colors.hard)}
              style={{ width: `${hard}%` }}
            />
          </SliderPrimitive.Track>
          <SliderPrimitive.Thumb
            aria-label="Easy share"
            className="block size-4 rounded-full border border-ring bg-background shadow-sm ring-ring/50 transition-shadow hover:ring-3 focus-visible:ring-3 focus-visible:outline-hidden"
          />
          <SliderPrimitive.Thumb
            aria-label="Easy plus medium share"
            className="block size-4 rounded-full border border-ring bg-background shadow-sm ring-ring/50 transition-shadow hover:ring-3 focus-visible:ring-3 focus-visible:outline-hidden"
          />
        </SliderPrimitive.Root>
      </div>
      <div className="grid gap-5 sm:grid-cols-3">
        {levels.map((level) => (
          <NumberInput
            key={level}
            id={`field-difficulty.${level}`}
            label={labels[level]}
            required
            suffix="%"
            value={level === 'easy' ? easy : level === 'medium' ? medium : hard}
            onChange={(next) => onChange(rebalance(value, level, next ?? 0))}
          />
        ))}
      </div>
      <FieldError message={error} />
    </div>
  )
}
