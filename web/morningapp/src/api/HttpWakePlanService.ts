import type {
  SaveWakePlanInput,
  WakePlan,
} from "../model/wakePlan";
import {
  getDefaultHttpErrorMessage,
  HttpError,
  isHttpError,
} from "./httpError";
import {
  httpClient,
  type HttpClient,
} from "./httpClient";
import type {
  WakePlanService,
} from "./wakePlanService";
import {
  parseWakePlanDto,
  toSaveWakePlanRequestDto,
} from "./wakePlanDto";

const CURRENT_WAKE_PLAN_PATH =
  "/api/v1/wake-plan";

export class HttpWakePlanService
  implements WakePlanService
{
  private readonly client: HttpClient;

  constructor(client: HttpClient) {
    this.client = client;
  }

  async getCurrent():
    Promise<WakePlan | null> {
    try {
      const response =
        await this.client.get<unknown>(
          CURRENT_WAKE_PLAN_PATH,
        );

      return parseWakePlanDto(response);
    } catch (error) {
      if (
        isHttpError(error) &&
        error.kind === "not_found"
      ) {
        return null;
      }

      throw error;
    }
  }

  async save(
    input: SaveWakePlanInput,
  ): Promise<WakePlan> {
    const request =
      toSaveWakePlanRequestDto(input);

    const response =
      await this.client.put<unknown>(
        CURRENT_WAKE_PLAN_PATH,
        request,
      );

    return parseWakePlanDto(response);
  }

  async cancel(
    wakePlanId: number,
  ): Promise<void> {
    validateWakePlanId(wakePlanId);

    await this.client.delete(
      `${CURRENT_WAKE_PLAN_PATH}/${wakePlanId}`,
);
}
}

export const httpWakePlanService =
    new HttpWakePlanService(httpClient);

function validateWakePlanId(
    wakePlanId: number,
): void {
    if (
        Number.isSafeInteger(wakePlanId) &&
        wakePlanId > 0
    ) {
        return;
    }

    throw new HttpError({
        kind: "validation",
        message:
            getDefaultHttpErrorMessage(
                "validation",
            ),
    });
}
