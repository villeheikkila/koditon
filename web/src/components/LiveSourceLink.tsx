type LiveSourceLinkProps = {
  available?: boolean | null
  url?: string | null
}

export default function LiveSourceLink({ available, url }: LiveSourceLinkProps) {
  if (!available || !url) return null
  return <a href={url} target="_blank" rel="noreferrer">Live source page</a>
}
