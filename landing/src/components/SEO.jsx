import { useEffect } from 'react'
import { useLocation } from 'react-router-dom'
import { pageSEO } from '../data/seo'
import { setSEO } from '../utils/seo'

export default function SEO({ config }) {
  const { pathname } = useLocation()

  useEffect(() => {
    const meta = config || pageSEO[pathname] || pageSEO['/']
    setSEO(meta)
  }, [pathname, config])

  return null
}
