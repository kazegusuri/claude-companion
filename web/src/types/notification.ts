export type NotificationSeverity = "info" | "warning" | "error" | "critical";
export type NotificationKind = "info" | "action_required" | "permission" | "system";
export type NotificationStatus = "unread" | "read" | "resolved" | "error";
export type NotificationSource = "agent" | "system" | "user" | "task";

export interface NotificationAction {
  type: "view" | "permit" | "deny" | "mute" | "custom";
  label: string;
  handler?: () => void | Promise<void>;
  messageId?: string;
  sessionId?: string;
}

export interface Notification {
  id: string;
  dedupeKey?: string | undefined;
  title: string;
  body: string;
  source: NotificationSource;
  sourceId?: string | undefined; // agentPID or taskId
  sourceName?: string | undefined; // agent name or task name
  severity: NotificationSeverity;
  kind: NotificationKind;
  status: NotificationStatus;
  actions?: NotificationAction[] | undefined;
  metadata?: Record<string, unknown> | undefined;
  createdAt: Date;
  readAt?: Date | undefined;
  expiresAt?: Date | undefined;
  messageId?: string | undefined; // for navigation to specific message
  sessionId?: string | undefined;
}

export interface NotificationFilter {
  kind?: NotificationKind | "all" | undefined;
  sources?: string[] | undefined;
  status?: NotificationStatus | undefined;
  severity?: NotificationSeverity | undefined;
}

export interface NotificationSettings {
  soundEnabled: boolean;
  vibrationEnabled: boolean;
  maxRetention: number;
  markOnOpen: boolean;
  autoExpireHours: number;
}
