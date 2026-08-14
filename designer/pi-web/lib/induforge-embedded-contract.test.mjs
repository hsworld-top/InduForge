import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import test from "node:test";

const themeSource = await readFile(new URL("../hooks/useTheme.ts", import.meta.url), "utf8");
const i18nSource = await readFile(new URL("../hooks/useI18n.tsx", import.meta.url), "utf8");
const layoutSource = await readFile(new URL("../app/layout.tsx", import.meta.url), "utf8");
const skillsSource = await readFile(new URL("../components/SkillsConfig.tsx", import.meta.url), "utf8");
const skillCheckSource = await readFile(new URL("../app/api/skills/check/route.ts", import.meta.url), "utf8");
const skillUpdateSource = await readFile(new URL("../app/api/skills/update/route.ts", import.meta.url), "utf8");

test("theme and locale are controlled only by the host", () => {
  assert.doesNotMatch(themeSource, /localStorage|matchMedia|toggleTheme|ThemePreference/);
  assert.doesNotMatch(i18nSource, /localStorage|resolveBrowserLocale|setLocale:/);
  assert.doesNotMatch(layoutSource, /pi-theme|localStorage/);
  assert.match(themeSource, /root\.classList\.toggle\("dark", next === "dark"\)/);
  assert.match(themeSource, /root\.dataset\.theme = next/);
  assert.match(themeSource, /root\.style\.colorScheme = next/);
  assert.match(themeSource, /theme = next;\s*applyDomTheme\(next\)/);
});

test("skills expose only the project scope", () => {
  assert.doesNotMatch(skillsSource, /return "global"|label: "path"/);
  assert.match(skillCheckSource, /body\.scope === "project" \? "project" : undefined/);
  assert.match(skillUpdateSource, /body\.scope === "project" \? "project" : undefined/);
  assert.doesNotMatch(skillCheckSource, /body\.scope === "global"/);
  assert.doesNotMatch(skillUpdateSource, /body\.scope === "global"/);
});
