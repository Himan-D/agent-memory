import { ImageResponse } from "next/og"

export const size = { width: 180, height: 180 }
export const contentType = "image/png"

export default function AppleIcon() {
  return new ImageResponse(
    (
      <div
        style={{
          width: "100%",
          height: "100%",
          display: "flex",
          alignItems: "center",
          justifyContent: "center",
          background: "linear-gradient(135deg, #6366f1 0%, #8b5cf6 100%)",
          borderRadius: 36,
        }}
      >
        <svg viewBox="0 0 128 128" width="120" height="120" fill="white">
          <circle cx="64" cy="64" r="12" />
          <circle cx="32" cy="40" r="7" opacity="0.9" />
          <circle cx="96" cy="40" r="7" opacity="0.9" />
          <circle cx="32" cy="88" r="7" opacity="0.9" />
          <circle cx="96" cy="88" r="7" opacity="0.9" />
          <circle cx="64" cy="24" r="5" opacity="0.7" />
          <circle cx="64" cy="104" r="5" opacity="0.7" />
        </svg>
      </div>
    ),
    { ...size }
  )
}
