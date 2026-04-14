import { useState } from 'react'
import { motion } from 'framer-motion'

const TALLY_WAITLIST_FORM_ID = 'wQ9aNz'
const TALLY_WIDGET_URL = 'https://tally.so/widgets/embed.js'

function Waitlist() {
  const [isOpen, setIsOpen] = useState(false)
  const [isLoading, setIsLoading] = useState(false)

  const openWaitlist = () => {
    setIsLoading(true)
    
    if (!document.querySelector(`script[src="${TALLY_WIDGET_URL}"]`)) {
      const script = document.createElement('script')
      script.src = TALLY_WIDGET_URL
      script.onload = () => {
        setIsLoading(false)
        setIsOpen(true)
        if (window.Tally) {
          window.Tally.openPopup(TALLY_WAITLIST_FORM_ID, {
            popupHeight: 500,
            onClose: () => setIsOpen(false)
          })
        }
      }
      script.onerror = () => {
        setIsLoading(false)
        console.error('Failed to load Tally widget')
      }
      document.head.appendChild(script)
    } else {
      setIsLoading(false)
      if (window.Tally) {
        window.Tally.openPopup(TALLY_WAITLIST_FORM_ID, {
          popupHeight: 500,
          onClose: () => setIsOpen(false)
        })
      }
    }
  }

  return (
    <button 
      onClick={openWaitlist}
      disabled={isLoading}
      className="waitlist-trigger"
    >
      {isLoading ? 'Opening...' : 'Join Waitlist'}
    </button>
  )
}

export default Waitlist
