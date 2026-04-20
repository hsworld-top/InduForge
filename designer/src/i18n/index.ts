import { createI18n } from "vue-i18n";
import { enMessages } from "./messages/en";
import { zhMessages } from "./messages/zh";

export const messages = {
  zh: zhMessages,
  en: enMessages,
} as const;

export const i18n = createI18n({
  legacy: false,
  locale: "zh",
  fallbackLocale: "zh",
  messages,
});
