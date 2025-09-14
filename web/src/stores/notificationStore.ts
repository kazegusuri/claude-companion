import { create } from "zustand";
import { persist } from "zustand/middleware";
import type { Notification, NotificationFilter, NotificationSettings } from "../types/notification";

interface NotificationStore {
  notifications: Notification[];
  filter: NotificationFilter;
  settings: NotificationSettings;
  isOpen: boolean;
  selectedId: string | null;

  // Actions
  pushNotifications: (notifications: Notification[]) => void;
  markAsRead: (ids: string[]) => void;
  markAsResolved: (id: string) => void;
  deleteNotification: (id: string) => void;
  clearAll: () => void;
  togglePanel: () => void;
  setFilter: (filter: NotificationFilter) => void;
  updateSettings: (settings: Partial<NotificationSettings>) => void;
  selectNotification: (id: string | null) => void;

  // Computed
  getUnreadCount: () => number;
  getFilteredNotifications: () => Notification[];
}

const DEFAULT_SETTINGS: NotificationSettings = {
  soundEnabled: true,
  vibrationEnabled: false,
  maxRetention: 100,
  markOnOpen: true,
  autoExpireHours: 72,
};

const STORAGE_KEY = "notification-center-v1";
const MAX_NOTIFICATIONS = 100;

export const useNotificationStore = create<NotificationStore>()(
  persist(
    (set, get) => ({
      notifications: [],
      filter: { kind: "all" },
      settings: DEFAULT_SETTINGS,
      isOpen: false,
      selectedId: null,

      pushNotifications: (newNotifications) => {
        set((state) => {
          const now = new Date();
          const processedNotifications = newNotifications.map((n) => ({
            ...n,
            createdAt: n.createdAt || now,
            status: n.status || ("unread" as const),
            expiresAt:
              n.expiresAt ||
              (n.severity === "info" && n.kind === "info"
                ? new Date(now.getTime() + state.settings.autoExpireHours * 60 * 60 * 1000)
                : undefined),
          }));

          let updated = [...state.notifications];

          processedNotifications.forEach((newNotif) => {
            if (newNotif.dedupeKey) {
              const existingIndex = updated.findIndex((n) => n.dedupeKey === newNotif.dedupeKey);
              if (existingIndex !== -1 && updated[existingIndex]) {
                // Update existing, preserve read status
                const existing = updated[existingIndex];
                updated[existingIndex] = {
                  ...newNotif,
                  status: existing.status,
                  readAt: existing.readAt || undefined,
                };
                return;
              }
            }
            updated.unshift(newNotif);
          });

          // Keep only MAX_NOTIFICATIONS
          if (updated.length > MAX_NOTIFICATIONS) {
            updated = updated.slice(0, MAX_NOTIFICATIONS);
          }

          // Remove expired notifications
          updated = updated.filter((n) => {
            if (!n.expiresAt) return true;
            return n.expiresAt > now;
          });

          return { notifications: updated };
        });
      },

      markAsRead: (ids) => {
        set((state) => ({
          notifications: state.notifications.map((n) =>
            ids.includes(n.id) && n.status === "unread"
              ? { ...n, status: "read" as const, readAt: new Date() as Date | undefined }
              : n,
          ),
        }));
      },

      markAsResolved: (id) => {
        set((state) => ({
          notifications: state.notifications.map((n) =>
            n.id === id ? { ...n, status: "resolved" } : n,
          ),
        }));
      },

      deleteNotification: (id) => {
        set((state) => ({
          notifications: state.notifications.filter((n) => n.id !== id),
        }));
      },

      clearAll: () => {
        set({ notifications: [] });
      },

      togglePanel: () => {
        set((state) => {
          const isOpening = !state.isOpen;

          // Mark visible notifications as read when opening
          if (isOpening && state.settings.markOnOpen) {
            const visibleNotifications = state
              .getFilteredNotifications()
              .filter((n) => n.status === "unread")
              .slice(0, 10)
              .map((n) => n.id);

            if (visibleNotifications.length > 0) {
              setTimeout(() => get().markAsRead(visibleNotifications), 500);
            }
          }

          return { isOpen: isOpening };
        });
      },

      setFilter: (filter) => {
        set({ filter });
      },

      updateSettings: (settings) => {
        set((state) => ({
          settings: { ...state.settings, ...settings },
        }));
      },

      selectNotification: (id) => {
        set({ selectedId: id });
      },

      getUnreadCount: () => {
        const state = get();
        return state.notifications.filter((n) => n.status === "unread").length;
      },

      getFilteredNotifications: () => {
        const state = get();
        const { filter, notifications } = state;

        return notifications.filter((n) => {
          if (filter.kind && filter.kind !== "all" && n.kind !== filter.kind) {
            return false;
          }
          if (filter.sources && filter.sources.length > 0 && !filter.sources.includes(n.source)) {
            return false;
          }
          if (filter.status && n.status !== filter.status) {
            return false;
          }
          if (filter.severity && n.severity !== filter.severity) {
            return false;
          }
          return true;
        });
      },
    }),
    {
      name: STORAGE_KEY,
      partialize: (state) => ({
        notifications: state.notifications.slice(0, MAX_NOTIFICATIONS),
        settings: state.settings,
      }),
    },
  ),
);
