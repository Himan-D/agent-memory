import { useState, useEffect, useMemo } from 'react'
import { BrowserRouter, Routes, Route, useLocation } from 'react-router-dom'
import { isBlogSubdomain, isStatusSubdomain } from './constants/blog'
import { ThemeProvider } from './context/ThemeContext'
import { AuthProvider } from './context/AuthContext'
import Navbar from './components/Navbar'
import Hero from './components/Hero'
import Metrics from './components/Metrics'
import HowItWorks from './components/HowItWorks'
import Features from './components/Features'
import AgentSection from './components/AgentSection'
import UseCases from './components/UseCases'
import Pricing from './components/Pricing'
import Blog from './components/Blog'
import CTA from './components/CTA'
import Footer from './components/Footer'
import BlogPost from './components/BlogPost'
import UseCasesPage from './pages/UseCasesPage'
import DocsPage from './pages/DocsPage'
import BlogPage from './pages/BlogPage'
import StatusPage from './pages/StatusPage'
import DemoPage from './pages/DemoPage'
import ForAgentsPage from './pages/ForAgentsPage'
import SEO from './components/SEO'

function Home() {
  const [loaded, setLoaded] = useState(false)

  useEffect(() => {
    setLoaded(true)
  }, [])

  return (
    <div className={`app ${loaded ? 'loaded' : ''}`}>
      <Hero />
      <Metrics />
      <HowItWorks />
      <Features />
      <AgentSection />
      <UseCases />
      <Pricing />
      <Blog />
      <CTA />
      <Footer />
    </div>
  )
}

function ScrollToHash() {
  const location = useLocation()

  useEffect(() => {
    if (location.hash) {
      const id = location.hash.replace('#', '')
      setTimeout(() => {
        const el = document.getElementById(id)
        if (el) {
          el.scrollIntoView({ behavior: 'smooth', block: 'start' })
        }
      }, 300)
    } else {
      window.scrollTo(0, 0)
    }
  }, [location])

  return null
}

function AppRoutes() {
  const onBlogDomain = useMemo(() => isBlogSubdomain(), [])
  const onStatusDomain = useMemo(() => isStatusSubdomain(), [])

  if (onBlogDomain) {
    return (
      <>
        <Navbar />
        <main className="main-content">
          <Routes>
            <Route path="/" element={<BlogPage />} />
            <Route path="/:slug" element={<BlogPost />} />
          </Routes>
        </main>
        <Footer />
      </>
    )
  }

  if (onStatusDomain) {
    return (
      <>
        <Navbar />
        <main className="main-content">
          <Routes>
            <Route path="/" element={<StatusPage />} />
            <Route path="*" element={<StatusPage />} />
          </Routes>
        </main>
      </>
    )
  }

  return (
    <>
      <Navbar />
      <main className="main-content">
        <Routes>
          <Route path="/" element={<Home />} />
          <Route path="/use-cases" element={<UseCasesPage />} />
          <Route path="/docs/*" element={<DocsPage />} />
          <Route path="/blog" element={<BlogPage />} />
          <Route path="/blog/:slug" element={<BlogPost />} />
          <Route path="/demo" element={<DemoPage />} />
          <Route path="/status" element={<StatusPage />} />
          <Route path="/for-agents" element={<ForAgentsPage />} />
        </Routes>
      </main>
    </>
  )
}

function App() {
  return (
    <ThemeProvider>
      <AuthProvider>
        <BrowserRouter>
          <SEO />
          <ScrollToHash />
          <div className="app">
            <AppRoutes />
          </div>
        </BrowserRouter>
      </AuthProvider>
    </ThemeProvider>
  )
}

export default App
