// UI regression harness: runs the real frontend against the in-memory backend in e2e/mock,
// walks through the main screens at several window sizes, languages and themes, and checks
// layout invariants a desktop app must keep. Screenshots go to e2e/screenshots for review.
//
//   npm run test:ui              all cases
//   npm run test:ui -- inbox     only cases whose name contains "inbox"
//
// It exits with 1 when an invariant is broken, so CI catches visual regressions such as the
// whole window scrolling, nested scrollbars or messages wider than the conversation.
import { mkdir, rm } from "node:fs/promises";
import { fileURLToPath } from "node:url";
import { chromium } from "playwright";
import { createServer } from "vite";

const here = fileURLToPath(new URL(".", import.meta.url));
const shots = `${here}screenshots`;
const filter = process.argv[2] ?? "";

// The app's minimum window (docs/UI.md §1), a common laptop and a large screen.
const sizes = [
  { width: 960, height: 640 },
  { width: 1280, height: 800 },
  { width: 1680, height: 1050 },
];

/**
 * Layout invariants, evaluated in the page. Returns a list of human-readable problems.
 * Kept in one function so it can be sent to the browser as is.
 */
function audit() {
  const problems = [];
  const describe = (el) => {
    const cls = typeof el.className === "string" ? el.className.split(/\s+/).slice(0, 4).join(".") : "";
    const label = el.getAttribute("aria-label") || el.getAttribute("role") || "";
    const text = (el.textContent || "").trim().slice(0, 40);
    return `<${el.tagName.toLowerCase()}${cls ? "." + cls : ""}${label ? ` [${label}]` : ""}> "${text}"`;
  };
  const scrollsY = (el) => {
    const s = getComputedStyle(el);
    return /(auto|scroll)/.test(s.overflowY) && el.scrollHeight > el.clientHeight + 1;
  };
  const clips = (el) => getComputedStyle(el).overflow !== "visible";

  // 1. The window itself never scrolls: everything fits and scrolls inside its own region.
  const doc = document.scrollingElement;
  if (doc.scrollHeight > innerHeight + 1 || doc.scrollWidth > innerWidth + 1) {
    const offenders = [];
    for (const el of document.body.querySelectorAll("*")) {
      const r = el.getBoundingClientRect();
      if (r.bottom <= innerHeight + 1 && r.right <= innerWidth + 1) continue;
      // An absolutely positioned element is only clipped by ancestors from its containing
      // block up: e.g. an sr-only label inside a scroll region whose containing block is
      // the page escapes that region and stretches the document.
      let p = el.parentElement;
      let clipped = false;
      let positioned = getComputedStyle(el).position;
      while (p && p !== document.body) {
        const style = getComputedStyle(p);
        const isContainingBlock =
          positioned === "static" || positioned === "relative" || positioned === "sticky" || style.position !== "static";
        if (isContainingBlock && clips(p)) {
          clipped = true;
          break;
        }
        if (positioned === "absolute" && style.position !== "static") positioned = "static";
        p = p.parentElement;
      }
      if (!clipped) offenders.push(describe(el));
      if (offenders.length >= 4) break;
    }
    problems.push(
      `document is ${doc.scrollWidth}×${doc.scrollHeight}, larger than the ${innerWidth}×${innerHeight} window` +
        (offenders.length ? `; escaping: ${offenders.join(", ")}` : ""),
    );
  }
  if (scrollX !== 0 || scrollY !== 0 || doc.scrollTop !== 0) {
    problems.push(`the window is scrolled to ${scrollX},${Math.max(scrollY, doc.scrollTop)}`);
  }

  // 2. No scroll region inside another one, and nothing scrolls sideways.
  const scrollers = [...document.querySelectorAll("*")].filter(scrollsY);
  for (const el of scrollers) {
    let p = el.parentElement;
    while (p && p !== document.body) {
      if (scrollsY(p)) {
        problems.push(`nested scroll: ${describe(el)} inside ${describe(p)}`);
        break;
      }
      p = p.parentElement;
    }
  }
  for (const el of document.querySelectorAll("*")) {
    const s = getComputedStyle(el);
    if (/(auto|scroll)/.test(s.overflowX) && el.scrollWidth > el.clientWidth + 1) {
      problems.push(`horizontal scroll in ${describe(el)} (${el.scrollWidth} > ${el.clientWidth})`);
    }
  }

  // 3. Messages stay inside the conversation.
  const log = document.querySelector('[role="log"]');
  if (log) {
    const box = log.getBoundingClientRect();
    for (const el of log.querySelectorAll("*")) {
      const r = el.getBoundingClientRect();
      if (r.width > 0 && (r.right > box.right + 1 || r.left < box.left - 1)) {
        problems.push(`wider than the conversation: ${describe(el)}`);
        break;
      }
    }
  }

  // 4. A text field that fits its text shows no scrollbar.
  for (const ta of document.querySelectorAll("textarea")) {
    const lines = ta.value.split("\n").length;
    if (ta.value.length < 40 && lines < 3 && ta.scrollHeight > ta.clientHeight + 1) {
      problems.push(`text field scrolls with short text: ${describe(ta)} (${ta.scrollHeight} > ${ta.clientHeight})`);
    }
  }
  return problems;
}

const failures = [];
let count = 0;

async function check(page, name) {
  count++;
  await page.waitForTimeout(150);
  const problems = await page.evaluate(audit);
  await page.screenshot({ path: `${shots}/${name}.png` });
  for (const p of problems) failures.push(`${name}: ${p}`);
  console.log(`${problems.length ? "FAIL" : "ok  "}  ${name}${problems.map((p) => `\n      - ${p}`).join("")}`);
}

/** Wheel over an element, as a user would with a mouse or trackpad. */
async function wheel(page, selector, dy, times = 1) {
  const box = await page.locator(selector).first().boundingBox();
  if (!box) return;
  await page.mouse.move(box.x + box.width / 2, box.y + box.height / 2);
  for (let i = 0; i < times; i++) {
    await page.mouse.wheel(0, dy);
    await page.waitForTimeout(40);
  }
}

const cases = [
  {
    name: "inbox",
    async run(page, tag) {
      await page.getByRole("listitem").first().waitFor();
      await check(page, `inbox-list-${tag}`);
      await wheel(page, '[aria-label="Chats"], [aria-label="Chats"] >> nth=0', 600, 8);
      await check(page, `inbox-list-scrolled-${tag}`);

      await page.getByRole("listitem").first().click();
      await page.locator('[role="log"]').waitFor();
      await page.waitForTimeout(300);
      await check(page, `chat-open-${tag}`);
      await wheel(page, '[role="log"]', -900, 25);
      await page.waitForTimeout(500);
      await check(page, `chat-scrolled-up-${tag}`);
      await wheel(page, '[role="log"]', 5000, 15);
      await wheel(page, '[role="log"]', 5000, 5);
      await check(page, `chat-past-bottom-${tag}`);

      const composer = page.locator("textarea");
      await composer.click();
      await composer.fill("one\ntwo\nthree\nfour\nfive\nsix\nseven\neight\nnine");
      await check(page, `composer-long-${tag}`);
      await composer.press("Enter");
      await page.waitForTimeout(300);
      await check(page, `chat-sent-${tag}`);

      await page.getByRole("listitem").nth(1).click();
      await page.waitForTimeout(400);
      await check(page, `group-open-${tag}`);
    },
  },
  {
    name: "settings",
    async run(page, tag) {
      await page.getByRole("listitem").first().waitFor();
      // The rail's last button is Settings; the second navigation lists its sections.
      await page.getByRole("navigation").first().getByRole("button").last().click();
      const sections = page.getByRole("navigation").nth(1).getByRole("button");
      await sections.first().waitFor();
      const names = ["general", "accounts", "privacy", "about"];
      for (let i = 0; i < names.length; i++) {
        await sections.nth(i + 1).click(); // the first button goes back to the inbox
        await check(page, `settings-${names[i]}-${tag}`);
      }
    },
  },
  {
    name: "dialogs",
    async run(page, tag) {
      await page.getByRole("listitem").first().waitFor();
      const banner = page.locator('[role="status"]').filter({ has: page.locator("button") }).first();
      await banner.locator("button").first().click();
      await check(page, `recovery-intro-${tag}`);
      await page.getByRole("dialog").locator("button").last().click();
      await check(page, `recovery-words-${tag}`);
      await page.keyboard.press("Escape");
      await page.keyboard.press(process.platform === "darwin" ? "Meta+n" : "Control+n");
      await check(page, `add-account-${tag}`);
    },
  },
  { name: "many-accounts", scenario: "many-accounts", async run(page, tag) { await check(page, `many-accounts-${tag}`); } },
  { name: "issues", scenario: "issues", async run(page, tag) { await check(page, `issues-${tag}`); } },
  { name: "empty", scenario: "empty", async run(page, tag) { await check(page, `empty-${tag}`); } },
  { name: "onboarding", scenario: "onboarding", async run(page, tag) { await check(page, `onboarding-${tag}`); } },
  { name: "locked", scenario: "locked", async run(page, tag) { await check(page, `locked-${tag}`); } },
];

await rm(shots, { recursive: true, force: true });
await mkdir(shots, { recursive: true });
const server = await createServer({ configFile: `${here}vite.config.ts`, logLevel: "error", server: { port: 5199 } });
await server.listen();
const browser = await chromium.launch(
  process.env.PRIVE_CHROMIUM ? { executablePath: process.env.PRIVE_CHROMIUM } : {},
);

try {
  for (const c of cases) {
    if (filter && !c.name.includes(filter)) continue;
    const variants = sizes.map((size, i) => ({ size, lang: i === 0 ? "es" : "en", theme: i === 1 ? "light" : "dark" }));
    for (const { size, lang, theme } of variants) {
      const tag = `${size.width}x${size.height}-${lang}-${theme}`;
      const page = await browser.newPage({ viewport: size, deviceScaleFactor: 1 });
      const errors = [];
      page.on("pageerror", (e) => errors.push(e.message));
      const scenario = c.scenario ?? "inbox";
      await page.goto(`http://localhost:5199/?scenario=${scenario}&lang=${lang}&theme=${theme}`);
      await page.waitForLoadState("networkidle");
      try {
        await c.run(page, tag);
      } catch (e) {
        failures.push(`${c.name} ${tag}: ${e.message.split("\n")[0]}`);
        console.log(`FAIL  ${c.name} ${tag}: ${e.message.split("\n")[0]}`);
        await page.screenshot({ path: `${shots}/${c.name}-${tag}-error.png` });
      }
      for (const e of errors) failures.push(`${c.name} ${tag}: page error: ${e}`);
      await page.close();
    }
  }
} finally {
  await browser.close();
  await server.close();
}

console.log(`\n${count} screens checked, ${failures.length} problems. Screenshots: e2e/screenshots/`);
process.exit(failures.length ? 1 : 0);
