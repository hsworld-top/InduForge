import fs from "fs";
import path from "path";
import { fileURLToPath } from "url";

const __dirname = path.dirname(fileURLToPath(import.meta.url));
const root = path.join(__dirname, "..");
const indexPath = path.join(root, "src/manifests/index.js");
const lines = fs.readFileSync(indexPath, "utf8").split(/\r?\n/);

const header = lines.slice(0, 78).join("\n");
const layout = lines.slice(80, 758).join("\n");
const diagramTextImage = lines.slice(759, 1075).join("\n");
const form = lines.slice(1075, 1468).join("\n");
const displayRest = lines.slice(1468, 1948).join("\n");
const chart = lines.slice(1948, 2257).join("\n");

const manifestDir = path.join(root, "src/manifests");
fs.writeFileSync(
  path.join(manifestDir, "manifest-registry.js"),
  `${header}\n\nexport default {\n  registerManifest,\n  getManifest,\n  getAllManifests,\n  getManifestsByCategory,\n};\n`,
);
fs.writeFileSync(
  path.join(manifestDir, "layout-manifests.js"),
  `import { registerManifest } from "./manifest-registry.js";\n\n${layout}\n`,
);
fs.writeFileSync(
  path.join(manifestDir, "form-manifests.js"),
  `import { registerManifest } from "./manifest-registry.js";\n\n${form}\n`,
);
fs.writeFileSync(
  path.join(manifestDir, "display-manifests.js"),
  `import { registerManifest } from "./manifest-registry.js";\n\n${diagramTextImage}\n${displayRest}\n`,
);
fs.writeFileSync(
  path.join(manifestDir, "chart-manifests.js"),
  `import { registerManifest } from "./manifest-registry.js";\n\n${chart}\n`,
);

const newIndex = `/**
 * 组件 Manifest 注册表 — 汇总入口
 * 侧向 import 触发各类 registerManifest
 */

export {
  registerManifest,
  getManifest,
  getAllManifests,
  getManifestsByCategory,
} from "./manifest-registry.js";

import "./layout-manifests.js";
import "./form-manifests.js";
import "./display-manifests.js";
import "./chart-manifests.js";

export { default } from "./manifest-registry.js";
`;

fs.writeFileSync(indexPath, newIndex);
console.log("split-manifests: OK");
