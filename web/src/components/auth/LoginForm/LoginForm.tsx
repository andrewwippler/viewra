import { useState } from 'react'
import { useForm } from '@tanstack/react-form'
import { useAuth } from '@/contexts'
import { FormInput, FormPasswordInput, FormApiError, FormSubmitButton } from '@/components/ui/Form'
import { getErrorMessage } from '@/lib/utils/error'
import { useWebOSInputNavigation, useWebOSKeyboardScroll } from '@/lib/hooks'
import type { LoginFormProps } from './LoginForm.types'
import { loginFormSchema, type LoginFormValues } from './LoginForm.schema'

const DEFAULT_VALUES: LoginFormValues = {
  username: '',
  password: '',
}

export const LoginForm = ({ onSuccess, variant = 'default' }: LoginFormProps) => {
  const { login } = useAuth()
  const [apiError, setApiError] = useState<string | null>(null)

  // Enable WebOS TV input navigation (up/down arrows act as Tab)
  useWebOSInputNavigation()
  useWebOSKeyboardScroll()

  const form = useForm({
    defaultValues: DEFAULT_VALUES,
    validators: {
      onChange: loginFormSchema,
    },
    onSubmit: async ({ value }) => {
      setApiError(null)

      try {
        await login(value.username, value.password)
        onSuccess()
      } catch (error) {
        setApiError(getErrorMessage(error, 'Login failed'))
      }
    },
  })

  const inputVariant = variant === 'glass' ? 'glass' : undefined

  return (
    <form
      onSubmit={(e) => {
        e.preventDefault()
        e.stopPropagation()
        form.handleSubmit()
      }}
      className="auth-form-enter space-y-5 tv-form-group"
    >
      <FormApiError error={apiError} className="mb-6" />

      <form.Field name="username">
        {(field) => (
          <div className="tv-field-wrapper">
            <FormInput
              field={field}
              label="Username"
              autoComplete="username"
              autoFocus
              tabIndex={1}
              variant={inputVariant}
              className=""
            />
          </div>
        )}
      </form.Field>

      <form.Field name="password">
        {(field) => (
          <div className="tv-field-wrapper">
            <FormPasswordInput
              field={field}
              label="Password"
              autoComplete="current-password"
              tabIndex={2}
              variant={inputVariant}
              className=""
            />
          </div>
        )}
      </form.Field>

      <div className="tv-field-wrapper">
        <FormSubmitButton
          form={form}
          requireDirty={false}
          fullWidth
          size="lg"
          className="mt-4 rounded-lg font-semibold "
        >
          Sign In
        </FormSubmitButton>
      </div>
    </form>
  )
}
