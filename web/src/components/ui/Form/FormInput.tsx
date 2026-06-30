import { Input, type InputProps } from '@/components/ui/Input'
import type { DeepKeys, DeepValue } from '@tanstack/react-form'
import { getFieldError, type AnyFieldApi } from './Form.types'

type FormInputProps<TFormData, TName extends DeepKeys<TFormData>> = {
  field: AnyFieldApi<TFormData, TName>
} & Omit<InputProps, 'value' | 'onChange' | 'onBlur' | 'error'>

/**
 * Form-connected Input component.
 * Automatically binds value, onChange, onBlur, and error display to TanStack Form field state.
 */
export const FormInput = <TFormData, TName extends DeepKeys<TFormData>>({
  field,
  className,
  ...props
}: FormInputProps<TFormData, TName>) => {
  const error = getFieldError(field)

  return (
    <Input
      value={field.state.value as string}
      onChange={(e) => field.handleChange(e.target.value as DeepValue<TFormData, TName>)}
      onBlur={field.handleBlur}
      error={error}
      className={className}
      {...props}
    />
  )
}
