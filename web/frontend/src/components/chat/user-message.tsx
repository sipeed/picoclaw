interface UserMessageProps {
  content: string
}

export function UserMessage({ content }: UserMessageProps) {
  return (
    <div className="flex w-full flex-col items-end gap-1.5">
      <div className="max-w-[70%] rounded-2xl rounded-tr-sm bg-violet-500 px-5 py-3 text-[15px] leading-relaxed text-white shadow-sm">
        {content}
      </div>
    </div>
  )
}
