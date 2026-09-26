// Builds the tracker: injects aod.json into the template's data placeholder.
//   node build.js  ->  aod-tracker.html
const fs = require("fs");
const data = fs.readFileSync("aod.json", "utf8");
JSON.parse(data);                                  // fail loudly on bad JSON
let html = fs.readFileSync("tracker.src.html", "utf8");
if (!html.includes("/*__AOD_DATA__*/{}")) throw new Error("data placeholder missing");
html = html.replace("/*__AOD_DATA__*/{}", data.trim());
fs.writeFileSync("aod-tracker.html", html);
console.log("built aod-tracker.html —", html.length, "bytes");
