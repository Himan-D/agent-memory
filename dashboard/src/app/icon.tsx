import { ImageResponse } from "next/og"

export const size = { width: 32, height: 32 }
export const contentType = "image/png"

export default function Icon() {
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
          borderRadius: 8,
        }}
      >
        <svg viewBox="0 0 128 128" width="22" height="22" fill="white">
          <circle cx="64" cy="64" r="12" />
          <circle cx="32" cy="40" r="7" opacity="0.9" />
          <circle cx="96" cy="40" r="7" opacity="0.9" />
          <circle cx="32" cy="88" r="7" opacity="0.9" />
          <circle cx="96" cy="88" r="7" opacity="0.9" />
        </svg>
      </div>
    ),
    { ...size }
  )
}
