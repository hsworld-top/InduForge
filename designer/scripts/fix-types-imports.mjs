import fs from "node:fs";
import path from "node:path";

const root = path.resolve("src");

function walk(dir) {
  for (const name of fs.readdirSync(dir, { withFileTypes: true })) {
    const p = path.join(dir, name.name);
    if (name.isDirectory()) {
      walk(p);
    } else if (/\.(js|vue|ts)$/.test(name.name)) {
      let c = fs.readFileSync(p, "utf8");
      const o = c;
      c = c.replaceAll("./types.js", "./types");
      c = c.replaceAll("../document/types.js", "../document/types");
      c = c.replaceAll("../../document/types.js", "../../document/types");
      c = c.replaceAll('"./document/types.js"', '"./document/types"');
      c = c.replaceAll("'./document/types.js'", "'./document/types'");
      c = c.replaceAll("'../editor-core/types.js'", "'../editor-core/document/types'");
      c = c.replaceAll('"../editor-core/types.js"', '"../editor-core/document/types"');
      if (c !== o) fs.writeFileSync(p, c);
    }
  }
}

walk(root);
console.log("done");
