import { useState } from 'react'
import { motion } from 'framer-motion'

const TALLY_FORM_ID = 'contact-sales'
const TALLY_WIDGET_URL = 'https://tally.so/widgets/embed.js'

function ContactForm() {
  const [isOpen, setIsOpen] = useState(false)
  const [isLoading, setIsLoading] = useState(false)

  const openTallyForm = () => {
    setIsLoading(true)
    
    // Load Tally widget script if not already loaded
    if (!document.querySelector(`script[src="${TALLY_WIDGET_URL}"]`)) {
      const script = document.createElement('script')
      script.src = TALLY_WIDGET_URL
      script.onload = () => {
        setIsLoading(false)
        setIsOpen(true)
        if (window.Tally) {
          window.Tally.openPopup(TALLY_FORM_ID, {
            popupHeight: 600,
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
        window.Tally.openPopup(TALLY_FORM_ID, {
          popupHeight: 600,
          onClose: () => setIsOpen(false)
        })
      }
    }
  }

  return (
    <>
      <button 
        onClick={openTallyForm}
        disabled={isLoading}
        className="contact-form-trigger"
      >
        {isLoading ? 'Opening...' : 'Contact Sales'}
      </button>

      {isOpen && (
        <div className="contact-form-overlay">
          <motion.div
            initial={{ opacity: 0 }}
            animate={{ opacity: 1 }}
            className="contact-form-backdrop"
            onClick={() => setIsOpen(false)}
          />
        </div>
      )}
    </>
  )
}

export default ContactForm
