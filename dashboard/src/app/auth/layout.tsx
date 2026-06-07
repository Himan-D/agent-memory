// Auth pages must not be statically cached at the CDN for a year.
export const dynamic = "force-dynamic";
export const revalidate = 0;

export default function AuthLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return children;
}
