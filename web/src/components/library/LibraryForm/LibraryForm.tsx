import { useState } from 'react'
import { useForm } from '@tanstack/react-form'
import { usePostApiLibraries } from '@/lib/api'
import { useInvalidateLibraries } from '@/lib/hooks/useInvalidateLibraries'
import { Button } from '@/components/ui'
import { FormInput, FormSelect, FormApiError, FormSubmitButton } from '@/components/ui/Form'
import { FilesystemBrowser } from '@/components/library/FilesystemBrowser'
import { getErrorMessage } from '@/lib/utils/error'
import { GithubComMantonxViewraInternalApplicationLibraryCreateLibraryRequestType } from '@/lib/api/generated/models'
import type { LibraryFormProps } from './LibraryForm.types'
import { libraryFormSchema, type LibraryFormValues } from './LibraryForm.schema'

const DEFAULT_VALUES: LibraryFormValues = {
  name: '',
  path: '',
  type: 'movies',
}

const LIBRARY_TYPE_OPTIONS = [
  { value: 'movies', label: 'Movies' },
  { value: 'tv', label: 'TV Shows' },
  { value: 'music', label: 'Music' },
  { value: 'live_tv', label: 'Live TV' },
]

const LibraryForm = ({ onCancel, onSuccess }: LibraryFormProps) => {
  const invalidateLibraries = useInvalidateLibraries()
  const createMutation = usePostApiLibraries()
  const [apiError, setApiError] = useState<string | null>(null)
  const [isBrowserOpen, setIsBrowserOpen] = useState(false)

  const form = useForm({
    defaultValues: DEFAULT_VALUES,
    validators: {
      onChange: libraryFormSchema,
    },
    onSubmit: async ({ value }) => {
      setApiError(null)

      try {
        await createMutation.mutateAsync({
          data: {
            name: value.name,
            path: value.path,
            type: value.type as GithubComMantonxViewraInternalApplicationLibraryCreateLibraryRequestType,
          },
        })

        invalidateLibraries()
        form.reset()
        onSuccess()
      } catch (error) {
        setApiError(getErrorMessage(error, 'Failed to create library'))
      }
    },
  })

  const handlePathSelect = (selectedPath: string) => {
    form.setFieldValue('path', selectedPath)
  }

  return (
    <form
      onSubmit={(e) => {
        e.preventDefault()
        e.stopPropagation()
        form.handleSubmit()
      }}
      className="space-y-4"
    >
      <FormApiError error={apiError} />

      <form.Field name="name">
        {(field) => (
          <FormInput field={field} label="Library Name" placeholder="My Movies" />
        )}
      </form.Field>

      <form.Field name="path">
        {(field) => (
          <div>
            <label className="block text-sm font-medium text-neutral-700 dark:text-neutral-300 mb-1">
              Path
            </label>
            <div className="flex gap-2">
              <FormInput
                field={field}
                placeholder="/media/movies"
                className="flex-1"
              />
              <form.Subscribe selector={(state) => state.isSubmitting}>
                {(isSubmitting) => (
                  <Button
                    type="button"
                    variant="secondary"
                    onClick={() => setIsBrowserOpen(true)}
                    disabled={isSubmitting as boolean}
                  >
                    Browse...
                  </Button>
                )}
              </form.Subscribe>
            </div>
          </div>
        )}
      </form.Field>

      <form.Field name="type">
        {(field) => (
          <div>
            <FormSelect field={field} label="Library Type" options={LIBRARY_TYPE_OPTIONS} />
            {field.state.value === 'live_tv' && (
              <p className="mt-1 text-sm text-amber-500">
                Point to a folder with .m3u files. Channel scan will auto-detect all playlists. You can also add an XMLTV URL in library settings for EPG data.
              </p>
            )}
          </div>
        )}
      </form.Field>

      <div className="flex gap-2 justify-end">
        <form.Subscribe selector={(state) => state.isSubmitting}>
          {(isSubmitting) => (
            <button
              type="button"
              onClick={onCancel}
              disabled={isSubmitting as boolean}
              className="px-4 py-2 text-sm font-medium rounded-lg bg-neutral-800 hover:bg-neutral-700 text-neutral-300 disabled:opacity-50 transition-colors"
            >
              Cancel
            </button>
          )}
        </form.Subscribe>
        <FormSubmitButton form={form} requireDirty={false}>
          Create Library
        </FormSubmitButton>
      </div>

      <FilesystemBrowser
        isOpen={isBrowserOpen}
        onClose={() => setIsBrowserOpen(false)}
        onSelect={handlePathSelect}
        initialPath={form.getFieldValue('path') || '/'}
      />
    </form>
  )
}

export { LibraryForm }
export type { LibraryFormProps } from './LibraryForm.types'
