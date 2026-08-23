import { useParams, useSearchParams } from 'react-router-dom'
import { ChatWindow } from '../components/chat/ChatWindow'

export function PublicChatPage() {
  const { slug } = useParams()
  const [params] = useSearchParams()
  const embed = params.get('embed') === '1'
  if (!slug) {
    return (
      <div className="flex h-full items-center justify-center px-6 text-center text-sm text-muted">
        Bot tidak ditemukan
      </div>
    )
  }
  return <ChatWindow slug={slug} variant={embed ? 'embed' : 'page'} />
}
