/// <reference types="vite/client" />

interface ImportMetaEnv {
    readonly VITE_WAKE_PLAN_SERVICE?:
        "mock" | "http";

    readonly VITE_API_BASE_URL?:
        string;

    readonly VITE_API_REQUEST_TIMEOUT_MS?:
        string;
}

interface ImportMeta {
    readonly env: ImportMetaEnv;
}