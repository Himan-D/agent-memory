import { useState, useEffect } from 'react'
import { BrowserRouter, Routes, Route } from 'react-router-dom'
import Navbar from './components/Navbar'
import Hero from './components/Hero'
import Features from './components/Features'
import CodeDemo from './components/CodeDemo'
import Metrics from './components/Metrics'
import UseCases from './components/UseCases'
import HowItWorks from './components/HowItWorks'
import Pricing from './components/Pricing'
import Blog from './components/Blog'
import CTA from './components/CTA'
import Footer from './components/Footer'
import BlogPost from './components/BlogPost'
import UseCasesPage from './pages/UseCasesPage'
import DocsPage from './pages/DocsPage'
import BlogPage from './pages/BlogPage'

function Home() {
  const [loaded, setLoaded] = useState(false)

  useEffect(() => {
    setLoaded(true)
  }, [])

  return (
    <div className={`app ${loaded ? 'loaded' : ''}`}>
      <div className="grain-overlay" />
      <Hero />
      <Metrics />
      <Features />
      <CodeDemo />
      <HowItWorks />
      <UseCases />
      <Pricing />
      <Blog />
      <CTA />
      <Footer />
    </div>
  )
}

function App() {
  return (
    <BrowserRouter>
      <div className="app">
        <Navbar />
        <main className="main-content">
          <Routes>
            <Route path="/" element={<Home />} />
            <Route path="/use-cases" element={<UseCasesPage />} />
            <Route path="/docs" element={<DocsPage />} />
            <Route path="/blog" element={<BlogPage />} />
            <Route path="/blog/:slug" element={<BlogPost />} />
          </Routes>
        </main>
      </div>
    </BrowserRouter>
  )
}

export default App