/**
 * Password validation result
 */
export interface PasswordValidationResult {
  valid: boolean
  errors: string[]
  strength: 'weak' | 'medium' | 'strong'
}

/**
 * Special characters allowed in passwords
 */
export const SPECIAL_CHARS = '!@#$%^&*()_+-=[]{}|;:,.<>?'

/**
 * Validate password strength
 * 纯自用：只要求最小长度，不强制大写/小写/数字/特殊字符。
 *
 * @param password Password to validate
 * @returns Validation result with errors and strength indicator
 */
export function validatePassword(password: string): PasswordValidationResult {
  const errors: string[] = []
  let score = 0

  // Check length（至少 6 位）
  if (password.length < 6) {
    errors.push('密码长度至少6位')
  } else {
    score++
    if (password.length >= 10) score++
    if (password.length >= 16) score++
  }

  // Determine strength
  let strength: 'weak' | 'medium' | 'strong'
  if (score <= 1) {
    strength = 'weak'
  } else if (score === 2) {
    strength = 'medium'
  } else {
    strength = 'strong'
  }

  return {
    valid: errors.length === 0,
    errors,
    strength,
  }
}

/**
 * Get password strength color for UI display
 * @param strength Password strength level
 * @returns Color string for Element Plus
 */
export function getStrengthColor(strength: 'weak' | 'medium' | 'strong'): string {
  switch (strength) {
    case 'weak':
      return '#F56C6C'
    case 'medium':
      return '#E6A23C'
    case 'strong':
      return '#67C23A'
    default:
      return '#909399'
  }
}

/**
 * Get password strength percentage for progress bar
 * @param strength Password strength level
 * @returns Percentage value (0-100)
 */
export function getStrengthPercentage(strength: 'weak' | 'medium' | 'strong'): number {
  switch (strength) {
    case 'weak':
      return 33
    case 'medium':
      return 66
    case 'strong':
      return 100
    default:
      return 0
  }
}

/**
 * Get password strength label for display
 * @param strength Password strength level
 * @returns Localized strength label
 */
export function getStrengthLabel(strength: 'weak' | 'medium' | 'strong'): string {
  switch (strength) {
    case 'weak':
      return '弱'
    case 'medium':
      return '中'
    case 'strong':
      return '强'
    default:
      return ''
  }
}
