import { appConfig } from "../config/appConfig";
import { telegramBridge } from "../telegram/telegram";
import {
  getDefaultHttpErrorMessage,
  getHttpErrorKind,
  HttpError,
} from "./httpError";

const TELEGRAM_AUTH_SCHEME = "tma";

export type HttpRequestOptions = Readonly<{
  signal?: AbortSignal;
  headers?: HeadersInit;
}>;

type HttpMethod =
  | "GET"
  | "PUT"
  | "DELETE";

type InternalRequestOptions =
  HttpRequestOptions &
    Readonly<{
      method: HttpMethod;
      body?: unknown;
      expectJson: boolean;
    }>;

export class HttpClient {
  private readonly baseUrl: string;

  private readonly timeoutMs: number;

  constructor(
    baseUrl: string,
    timeoutMs: number,
  ) {
    this.baseUrl = baseUrl;
    this.timeoutMs = timeoutMs;
  }

  get<T>(
    path: string,
    options: HttpRequestOptions = {},
  ): Promise<T> {
    return this.request<T>(path, {
      ...options,
      method: "GET",
      expectJson: true,
    });
  }

  put<T>(
    path: string,
    body: unknown,
    options: HttpRequestOptions = {},
  ): Promise<T> {
    return this.request<T>(path, {
      ...options,
      method: "PUT",
      body,
      expectJson: true,
    });
  }

  delete(
    path: string,
    options: HttpRequestOptions = {},
  ): Promise<void> {
    return this.request<void>(path, {
      ...options,
      method: "DELETE",
      expectJson: false,
    });
  }

  private async request<T>(
    path: string,
    options: InternalRequestOptions,
  ): Promise<T> {
    const controller = new AbortController();

    let didTimeout = false;

    const timeoutId = globalThis.setTimeout(
      () => {
        didTimeout = true;
        controller.abort();
      },
      this.timeoutMs,
    );

    const removeExternalAbortListener =
      forwardAbortSignal(
        options.signal,
        controller,
      );

    try {
      const response = await fetch(
        buildRequestUrl(
          this.baseUrl,
          path,
        ),
        {
          method: options.method,
          headers: buildRequestHeaders(
            options.headers,
            options.body !== undefined,
          ),
          body: serializeRequestBody(
            options.body,
          ),
          signal: controller.signal,
          credentials: "omit",
          cache: "no-store",
        },
      );

      const responseBody =
        await readResponseBody(response);

      if (!response.ok) {
        throw createHttpStatusError(
          response.status,
          responseBody,
        );
      }

      if (!options.expectJson) {
        return undefined as T;
      }

      if (responseBody === null) {
        throw new HttpError({
          kind: "invalid_response",
          status: response.status,
          message:
            getDefaultHttpErrorMessage(
              "invalid_response",
            ),
        });
      }

      return responseBody as T;
    } catch (error) {
      if (error instanceof HttpError) {
        throw error;
      }

      if (didTimeout) {
        throw new HttpError({
          kind: "timeout",
          message:
            getDefaultHttpErrorMessage(
              "timeout",
            ),
          originalError: error,
        });
      }

      if (
        options.signal?.aborted ||
        controller.signal.aborted
      ) {
        throw new HttpError({
          kind: "aborted",
          message:
            getDefaultHttpErrorMessage(
              "aborted",
            ),
          originalError: error,
        });
      }

      if (error instanceof TypeError) {
        throw new HttpError({
          kind: "network",
          message:
            getDefaultHttpErrorMessage(
              "network",
            ),
          originalError: error,
        });
      }

      throw new HttpError({
        kind: "unknown",
        message:
          getDefaultHttpErrorMessage(
            "unknown",
          ),
        originalError: error,
      });
    } finally {
      globalThis.clearTimeout(timeoutId);

      removeExternalAbortListener();
    }
  }
}

export const httpClient = new HttpClient(
  appConfig.apiBaseUrl,
  appConfig.apiRequestTimeoutMs,
);

function buildRequestUrl(
  baseUrl: string,
  path: string,
): string {
  const normalizedPath =
    path.startsWith("/")
      ? path
      : `/${path}`;

  return `${baseUrl}${normalizedPath}`;
}

function buildRequestHeaders(
  customHeaders: HeadersInit | undefined,
  hasBody: boolean,
): Headers {
  const headers = new Headers(
    customHeaders,
  );

  if (!headers.has("Accept")) {
    headers.set(
      "Accept",
      "application/json",
    );
  }

  if (
    hasBody &&
    !headers.has("Content-Type")
  ) {
    headers.set(
      "Content-Type",
      "application/json",
    );
  }

  const initData =
    telegramBridge.getInitData().trim();

  if (initData.length > 0) {
    headers.set(
      "Authorization",
      `${TELEGRAM_AUTH_SCHEME} ${initData}`,
    );
  }

  return headers;
}

function serializeRequestBody(
  body: unknown,
): string | undefined {
  if (body === undefined) {
    return undefined;
  }

  try {
    return JSON.stringify(body);
  } catch (error) {
    throw new HttpError({
      kind: "validation",
      message:
        "The request body could not be serialized.",
      originalError: error,
    });
  }
}

async function readResponseBody(
  response: Response,
): Promise<unknown> {
  if (response.status === 204) {
    return null;
  }

  const text = await response.text();

  if (text.trim().length === 0) {
    return null;
  }

  const contentType =
    response.headers
      .get("content-type")
      ?.toLowerCase() ?? "";

  if (
    !contentType.includes(
      "application/json",
    )
  ) {
    return text;
  }

  try {
    return JSON.parse(text) as unknown;
  } catch (error) {
    throw new HttpError({
      kind: "invalid_response",
      status: response.status,
      responseBody: text,
      message:
        getDefaultHttpErrorMessage(
          "invalid_response",
        ),
      originalError: error,
    });
  }
}

function createHttpStatusError(
  status: number,
  responseBody: unknown,
): HttpError {
  const kind =
    getHttpErrorKind(status);

  const fallbackMessage =
    getDefaultHttpErrorMessage(kind);

  return new HttpError({
    kind,
    status,
    responseBody,
    message: readServerErrorMessage(
      responseBody,
      fallbackMessage,
    ),
  });
}

function readServerErrorMessage(
  responseBody: unknown,
  fallback: string,
): string {
  if (
    typeof responseBody === "object" &&
    responseBody !== null
  ) {
    const body = responseBody as {
      message?: unknown;
      error?: unknown;
    };

    if (
      typeof body.message === "string" &&
      body.message.trim().length > 0
    ) {
      return body.message.trim();
    }

    if (
      typeof body.error === "string" &&
      body.error.trim().length > 0
    ) {
      return body.error.trim();
    }
  }

  if (
    typeof responseBody === "string"
  ) {
    const message =
      responseBody.trim();

    if (
      message.length > 0 &&
      message.length <= 500 &&
      !message.startsWith("<")
    ) {
      return message;
    }
  }

  return fallback;
}

function forwardAbortSignal(
  externalSignal: AbortSignal | undefined,
  controller: AbortController,
): () => void {
  if (!externalSignal) {
    return () => undefined;
  }

  if (externalSignal.aborted) {
    controller.abort();

    return () => undefined;
  }

  function handleExternalAbort() {
    controller.abort();
  }

  externalSignal.addEventListener(
    "abort",
    handleExternalAbort,
    {
      once: true,
    },
  );

  return () => {
    externalSignal.removeEventListener(
      "abort",
      handleExternalAbort,
    );
  };
}
