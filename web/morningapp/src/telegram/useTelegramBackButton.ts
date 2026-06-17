import {
    useEffect,
    useRef,
} from "react";

import {
    telegramBridge,
} from "./telegram";

type UseTelegramBackButtonOptions = {
    isVisible: boolean;
    onBack: () => void;
};

export function useTelegramBackButton({
                                          isVisible,
                                          onBack,
                                      }: UseTelegramBackButtonOptions): void {
    const onBackRef = useRef(onBack);

    useEffect(() => {
        onBackRef.current = onBack;
    }, [onBack]);

    useEffect(() => {
        if (!isVisible) {
            telegramBridge.setBackButtonHandler(null);

            return;
        }

        function handleTelegramBackButton() {
            onBackRef.current();
        }

        telegramBridge.setBackButtonHandler(
            handleTelegramBackButton,
        );

        return () => {
            telegramBridge.setBackButtonHandler(null);
        };
    }, [isVisible]);
}