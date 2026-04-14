import type { ChatAttachment } from "@/store/chat"

interface UserMessageProps {
  content: string
  attachments?: ChatAttachment[]
}

export function UserMessage({ content, attachments = [] }: UserMessageProps) {
  const hasText = content.trim().length > 0
  const imageAttachments = attachments.filter(
    (attachment) => attachment.type === "image",
  )

  return (
    <div className="flex w-full flex-col items-end gap-1.5">
      {imageAttachments.length > 0 && (
        <div className="flex max-w-[70%] flex-wrap justify-end gap-2">
          {imageAttachments.map((attachment, index) => (
            <img
              key={`${attachment.url}-${index}`}
              src={attachment.url}
              alt={attachment.filename || "Uploaded image"}
              className="max-h-72 max-w-full object-cover"
            />
          ))}
        </div>
      )}

      {hasText && (
        <div className="max-w-[70%] rounded-2xl rounded-tr-sm bg-violet-500 px-5 py-3 text-[15px] leading-relaxed wrap-break-word whitespace-pre-wrap text-white shadow-sm">
          {content}
        </div>
      )}
    </div>
  )
}
