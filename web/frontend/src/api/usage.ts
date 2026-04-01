// Usage API — fetch usage statistics

import { launcherFetch } from "@/api/http"

export interface UsageStats {
  model_name: string
  model: string
  provider?: string
  message_count: number
  input_tokens: number
  output_tokens: number
  total_tokens: number
  estimated_cost: number
  currency: string
  session_count: number
}

export interface UsageResponse {
  stats: UsageStats[]
  total_input_tokens: number
  total_output_tokens: number
  total_tokens: number
  total_message_count: number
  total_estimated_cost: number
  currency: string
  date_range: string
}

export async function getUsage(
  startDate?: string,
  endDate?: string,
): Promise<UsageResponse> {
  const params = new URLSearchParams()
  if (startDate) {
    params.set("start_date", startDate)
  }
  if (endDate) {
    params.set("end_date", endDate)
  }

  const queryString = params.toString()
  const url = `/api/usage${queryString ? `?${queryString}` : ""}`

  const res = await launcherFetch(url)
  if (!res.ok) {
    throw new Error(`Failed to fetch usage: ${res.status}`)
  }
  return res.json()
}
