export interface Message {
  role: "system" | "user" | "assistant" | "tool";
  content: string;
}

export interface Tool {
  name: string;
  description: string;
  parameters: string;
}

export interface LLMCall {
  iteration: number;
  model: string;
  messages_count: number;
  tools_count: number;
  max_tokens: number;
  system_prompt_len?: number;
  temperature?: string;
  content_len: number | null;
  tool_calls_count: number | null;
  has_reasoning: boolean | null;
  timestamp: string;
  messages: Message[];
  tools: Tool[];
}

export interface ToolExec {
  tool: string;
  args_count: number;
  duration_ms: number | null;
  is_error: boolean | null;
  for_llm_len: number | null;
  timestamp: string;
}

export interface Turn {
  turn_id: string;
  timestamp: string;
  channel: string;
  chat_id: string;
  sender_id: string;
  session_key: string;
  user_len: number;
  status: string;
  duration_ms: number | null;
  iterations: number;
  llm_calls: LLMCall[];
  tool_execs: ToolExec[];
}
