export const appViews = {
    tonight: "tonight",
    schedule: "schedule",
} as const;

export type AppView =
    (typeof appViews)[keyof typeof appViews];

export const defaultAppView: AppView =
    appViews.tonight;