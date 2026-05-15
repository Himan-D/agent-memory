import { urlFor } from '../lib/sanity'

export const components = {
  types: {
    image: ({ value }) => {
      if (!value?.asset?._ref) return null
      return (
        <figure className="blog-image">
          <img src={urlFor(value).width(800).fit('max').url()} alt={value.alt || ''} loading="lazy" />
          {value.caption && <figcaption>{value.caption}</figcaption>}
        </figure>
      )
    },
    code: ({ value }) => {
      if (!value?.code) return null
      return (
        <div className="blog-code">
          {value.language && <span className="blog-code-lang">{value.language}</span>}
          <pre><code className={`language-${value.language || 'text'}`}>{value.code}</code></pre>
        </div>
      )
    }
  },
  marks: {
    link: ({ value, children }) => (
      <a href={value?.href} target="_blank" rel="noopener noreferrer">{children}</a>
    ),
    strong: ({ children }) => <strong>{children}</strong>,
    em: ({ children }) => <em>{children}</em>,
    code: ({ children }) => <code className="inline-code">{children}</code>
  },
  block: {
    h1: ({ children }) => <h1>{children}</h1>,
    h2: ({ children }) => <h2>{children}</h2>,
    h3: ({ children }) => <h3>{children}</h3>,
    h4: ({ children }) => <h4>{children}</h4>,
    blockquote: ({ children }) => <blockquote>{children}</blockquote>,
    normal: ({ children }) => <p>{children}</p>
  },
  list: {
    bullet: ({ children }) => <ul>{children}</ul>,
    number: ({ children }) => <ol>{children}</ol>
  },
  listItem: {
    bullet: ({ children }) => <li>{children}</li>,
    number: ({ children }) => <li>{children}</li>
  }
}
