import {
  DndContext,
  KeyboardSensor,
  PointerSensor,
  closestCenter,
  useSensor,
  useSensors,
  type DragEndEvent,
} from '@dnd-kit/core'
import { restrictToParentElement, restrictToVerticalAxis } from '@dnd-kit/modifiers'
import {
  SortableContext,
  sortableKeyboardCoordinates,
  useSortable,
  verticalListSortingStrategy,
} from '@dnd-kit/sortable'
import { CSS } from '@dnd-kit/utilities'
import { GripVerticalIcon, PlusIcon, Trash2Icon } from 'lucide-react'
import { useRef, useState, type PointerEvent, type ReactNode } from 'react'
import {
  Controller,
  useFieldArray,
  useFormContext,
  type ArrayPath,
  type FieldArray,
  type FieldValues,
  type Path,
} from 'react-hook-form'

import { Button } from '@/components/ui/button'
import { cn } from '@/lib/utils'

import { errorAt, FieldError, FieldLabel } from './field'

export interface RowColumn {
  key: string
  label: string
  placeholder?: string
  type?: 'text' | 'textarea' | 'date' | 'radio'
  width?: number
}

interface RowsFieldProps<T extends FieldValues> {
  name: ArrayPath<T>
  label: string
  help?: string
  required?: boolean
  columns: RowColumn[]
  emptyRow: FieldArray<T, ArrayPath<T>>
  addLabel: string
  minRows?: number
}

const HANDLE_WIDTH = 32
const REMOVE_WIDTH = 40
const MIN_COLUMN_WIDTH = 72

const cellClass =
  'block w-full min-w-0 rounded-none border-0 bg-transparent px-2 py-2 text-sm outline-none placeholder:text-muted-foreground focus-visible:bg-muted/60 aria-invalid:bg-destructive/10'

// RowsField edits a repeating table. Rows are reordered by dragging the
// handle, or with the keyboard by focusing the handle, pressing space,
// and using the arrow keys. Column dividers in the header drag to
// resize. A radio column marks exactly one row.
export function RowsField<T extends FieldValues>({
  name,
  label,
  help,
  required,
  columns,
  emptyRow,
  addLabel,
  minRows = 0,
}: RowsFieldProps<T>) {
  const { control, register, formState, setValue } = useFormContext<T>()
  const { fields, append, remove, move } = useFieldArray({ control, name })
  const listError = errorAt(formState.errors, name)
  const sensors = useSensors(
    useSensor(PointerSensor, { activationConstraint: { distance: 4 } }),
    useSensor(KeyboardSensor, { coordinateGetter: sortableKeyboardCoordinates }),
  )
  const { widths, tableRef, startResize } = useColumnWidths(columns)
  // The array path joined with an index and a column key is a valid
  // field path. react-hook-form cannot express that relationship in its
  // types, so the path is asserted here.
  const cellPath = (index: number, key: string) => `${name}.${index}.${key}` as Path<T>

  function markRow(index: number, key: string) {
    fields.forEach((_, rowIndex) => {
      setValue(cellPath(rowIndex, key), (rowIndex === index) as never, {
        shouldDirty: true,
      })
    })
  }

  function onDragEnd(event: DragEndEvent) {
    const { active, over } = event
    if (over === null || active.id === over.id) return
    const from = fields.findIndex((row) => row.id === active.id)
    const to = fields.findIndex((row) => row.id === over.id)
    if (from !== -1 && to !== -1) move(from, to)
  }

  const tableWidth =
    widths === null
      ? undefined
      : widths.reduce((sum, width) => sum + width, HANDLE_WIDTH + REMOVE_WIDTH)

  return (
    <div className="flex flex-col gap-2">
      <FieldLabel required={required} help={help}>
        {label}
      </FieldLabel>
      <div className="overflow-x-auto rounded-lg border">
        <DndContext
          sensors={sensors}
          collisionDetection={closestCenter}
          modifiers={[restrictToVerticalAxis, restrictToParentElement]}
          onDragEnd={onDragEnd}
        >
          <table
            ref={tableRef}
            className="min-w-full table-fixed border-collapse text-sm"
            style={tableWidth === undefined ? undefined : { width: tableWidth }}
          >
            <colgroup>
              <col style={{ width: HANDLE_WIDTH }} />
              {columns.map((column, index) => (
                <col
                  key={column.key}
                  style={{ width: widths?.[index] ?? column.width }}
                />
              ))}
              <col style={{ width: REMOVE_WIDTH }} />
            </colgroup>
            <thead className="bg-muted/50 text-left text-xs text-muted-foreground">
              <tr>
                <th className="py-1.5">
                  <span className="sr-only">Reorder</span>
                </th>
                {columns.map((column, index) => (
                  <th
                    key={column.key}
                    className="relative border-l px-2 py-1.5 font-medium"
                  >
                    <span className="block truncate">{column.label}</span>
                    {index < columns.length - 1 ? (
                      <span
                        role="separator"
                        aria-orientation="vertical"
                        aria-label={`Resize ${column.label} column`}
                        onPointerDown={(event) => startResize(index, event)}
                        className="absolute inset-y-0 -right-1 z-10 w-2 cursor-col-resize touch-none hover:bg-ring/60"
                      />
                    ) : null}
                  </th>
                ))}
                <th className="border-l py-1.5">
                  <span className="sr-only">Remove</span>
                </th>
              </tr>
            </thead>
            <SortableContext
              items={fields.map((row) => row.id)}
              strategy={verticalListSortingStrategy}
            >
              <tbody>
                {fields.map((row, index) => (
                  <SortableRow key={row.id} id={row.id} index={index}>
                    {columns.map((column) => {
                      const path = cellPath(index, column.key)
                      const error = errorAt(formState.errors, path)
                      const id = `field-${path}`
                      const ariaLabel = `${column.label}, row ${index + 1}`
                      return (
                        <td key={column.key} className="h-full border-l p-0 align-top">
                          {column.type === 'radio' ? (
                            <Controller
                              control={control}
                              name={path}
                              render={({ field }) => (
                                <input
                                  id={id}
                                  type="radio"
                                  name={`${name}-${column.key}`}
                                  aria-label={ariaLabel}
                                  checked={field.value === true}
                                  onChange={() => markRow(index, column.key)}
                                  className="m-2 mt-2.5 size-4 accent-foreground"
                                />
                              )}
                            />
                          ) : column.type === 'textarea' ? (
                            <textarea
                              id={id}
                              rows={1}
                              placeholder={column.placeholder}
                              aria-label={ariaLabel}
                              aria-invalid={error ? true : undefined}
                              className={cn(
                                cellClass,
                                'h-full max-h-40 min-h-full resize-none overflow-y-auto field-sizing-content',
                              )}
                              {...register(path)}
                            />
                          ) : (
                            <input
                              id={id}
                              type={column.type === 'date' ? 'date' : 'text'}
                              placeholder={column.placeholder}
                              aria-label={ariaLabel}
                              aria-invalid={error ? true : undefined}
                              className={cn(cellClass, 'h-full')}
                              {...register(path)}
                            />
                          )}
                          {error ? (
                            <div className="px-2 pb-1">
                              <FieldError message={error} />
                            </div>
                          ) : null}
                        </td>
                      )
                    })}
                    <td className="border-l p-0 align-top">
                      <div className="flex justify-center py-1">
                        <Button
                          type="button"
                          variant="ghost"
                          size="icon-sm"
                          aria-label={`Remove row ${index + 1}`}
                          disabled={fields.length <= minRows}
                          onClick={() => remove(index)}
                        >
                          <Trash2Icon />
                        </Button>
                      </div>
                    </td>
                  </SortableRow>
                ))}
                {fields.length === 0 ? (
                  <tr className="border-t">
                    <td
                      colSpan={columns.length + 2}
                      className="px-2 py-3 text-center text-muted-foreground"
                    >
                      No rows yet.
                    </td>
                  </tr>
                ) : null}
              </tbody>
            </SortableContext>
          </table>
        </DndContext>
      </div>
      <div>
        <Button
          type="button"
          variant="outline"
          size="sm"
          onClick={() => append(emptyRow)}
        >
          <PlusIcon data-icon="inline-start" />
          {addLabel}
        </Button>
      </div>
      <FieldError message={listError} />
    </div>
  )
}

// useColumnWidths tracks column widths in pixels once a divider has
// been dragged. Until then the browser lays the columns out itself.
function useColumnWidths(columns: RowColumn[]) {
  const tableRef = useRef<HTMLTableElement>(null)
  const [widths, setWidths] = useState<number[] | null>(null)

  function measure(): number[] {
    const headers = tableRef.current?.querySelectorAll('thead th') ?? []
    return columns.map((column, index) => {
      const header = headers[index + 1]
      return header instanceof HTMLElement ? header.offsetWidth : (column.width ?? 160)
    })
  }

  function startResize(index: number, event: PointerEvent<HTMLElement>) {
    event.preventDefault()
    const handle = event.currentTarget
    const startX = event.clientX
    const startWidths = widths ?? measure()
    handle.setPointerCapture(event.pointerId)

    function onMove(moveEvent: globalThis.PointerEvent) {
      const next = [...startWidths]
      const delta = moveEvent.clientX - startX
      next[index] = Math.max(MIN_COLUMN_WIDTH, (startWidths[index] ?? 0) + delta)
      setWidths(next)
    }

    function onUp() {
      handle.removeEventListener('pointermove', onMove)
      handle.removeEventListener('pointerup', onUp)
      handle.removeEventListener('pointercancel', onUp)
    }

    handle.addEventListener('pointermove', onMove)
    handle.addEventListener('pointerup', onUp)
    handle.addEventListener('pointercancel', onUp)
  }

  return { widths, tableRef, startResize }
}

// SortableRow is a table row with a drag handle in its first cell. The
// one-pixel height lets cell contents stretch to the row's full height.
function SortableRow({
  id,
  index,
  children,
}: {
  id: string
  index: number
  children: ReactNode
}) {
  const {
    attributes,
    listeners,
    setNodeRef,
    setActivatorNodeRef,
    transform,
    transition,
    isDragging,
  } = useSortable({ id })

  return (
    <tr
      ref={setNodeRef}
      style={{ transform: CSS.Transform.toString(transform), transition, height: 1 }}
      className={cn('border-t bg-card', isDragging && 'relative z-10 shadow-md')}
    >
      <td className="p-0 align-top">
        <div className="flex justify-center py-1.5">
          <button
            type="button"
            ref={setActivatorNodeRef}
            aria-label={`Drag to reorder row ${index + 1}`}
            className="inline-flex cursor-grab touch-none rounded p-1 text-muted-foreground hover:bg-muted hover:text-foreground active:cursor-grabbing"
            {...attributes}
            {...listeners}
          >
            <GripVerticalIcon className="size-4" />
          </button>
        </div>
      </td>
      {children}
    </tr>
  )
}
