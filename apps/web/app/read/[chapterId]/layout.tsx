import type { Metadata } from "next";

export const metadata: Metadata = {
  title: "Reading - Miru",
};

export default function ReadLayout({
  children,
}: Readonly<{
  children: React.ReactNode;
}>) {
  return (
    <>
      <meta name="apple-mobile-web-app-capable" content="yes" />
      <meta name="apple-mobile-web-app-status-bar-style" content="black-translucent" />
      <meta name="mobile-web-app-capable" content="yes" />
      {children}
    </>
  );
}
