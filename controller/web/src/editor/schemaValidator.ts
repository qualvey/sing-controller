import Ajv2020 from 'ajv/dist/2020'
import addFormats from 'ajv-formats'
import type { ValidateFunction } from 'ajv'
import schemaJson from '../../schema.json'

let validateFn: ValidateFunction | null = null

export function getSingBoxValidator(): ValidateFunction | null {
  if (validateFn) return validateFn
  try {
    const ajv = new Ajv2020({
      allErrors: true,
      strict: false,
      validateFormats: true,
    })
    addFormats(ajv)
    validateFn = ajv.compile(schemaJson)
    return validateFn
  } catch (err) {
    console.warn('[SchemaValidator] Failed to compile sing-box schema:', err)
    return null
  }
}
