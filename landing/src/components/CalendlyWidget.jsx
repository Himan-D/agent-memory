import { useEffect } from 'react'

const CALENDLY_URL = 'https://calendly.com/hystersis/30min'
const CALENDLY_SCRIPT = 'https://assets.calendly.com/assets/external/widget.js'

function CalendlyWidget() {
  useEffect(() => {
    const script = document.createElement('script')
    script.src = CALENDLY_SCRIPT
    script.async = true
    document.head.appendChild(script)
    return () => {
      document.head.removeChild(script)
    }
  }, [])

  return (
    <div className="calendly-wrapper">
      <div
        className="calendly-inline-widget"
        data-url={CALENDLY_URL}
        style={{ minWidth: '320px', height: '700px' }}
      />
    </div>
  )
}

export default CalendlyWidget
