function Icon({ children, className }) {
  return (
    <svg
      className={className}
      width="18"
      height="18"
      viewBox="0 0 24 24"
      fill="none"
      stroke="currentColor"
      strokeWidth="2"
      strokeLinecap="round"
      strokeLinejoin="round"
      aria-hidden="true"
    >
      {children}
    </svg>
  )
}

export function PlusIcon() {
  return (
    <Icon>
      <path d="M12 5v14M5 12h14" />
    </Icon>
  )
}

export function RefreshIcon({ className }) {
  return (
    <Icon className={className}>
      <path d="M20 11a8 8 0 0 0-14.3-4.9L3 9M3 4v5h5M4 13a8 8 0 0 0 14.3 4.9L21 15M21 20v-5h-5" />
    </Icon>
  )
}

export function CloseIcon() {
  return (
    <Icon>
      <path d="M6 6l12 12M18 6L6 18" />
    </Icon>
  )
}

export function SwapIcon() {
  return (
    <Icon>
      <path d="M7 4L3 8l4 4M3 8h14M17 20l4-4-4-4M21 16H7" />
    </Icon>
  )
}

export function AlertIcon() {
  return (
    <Icon>
      <circle cx="12" cy="12" r="9" />
      <path d="M12 8v5M12 16.5v.01" />
    </Icon>
  )
}

export function ChartIcon() {
  return (
    <Icon>
      <path d="M4 17l6-6 4 4 6-8" />
    </Icon>
  )
}
