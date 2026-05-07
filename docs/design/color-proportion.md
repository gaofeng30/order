<design_rules>
# UI Design System: Business Blue & Semantic Green

You are an expert UI designer. When generating or modifying UI code, you MUST strictly adhere to the following color craft and constraint rules. 

## 1. Palette Allocation (The 90-7-3 Rule)
- **Neutrals (90%)**: Backgrounds, surfaces, borders, and typography.
- **Primary Accent - Business Blue (7%)**: `var(--accent)`. Used ONLY for Primary CTAs, active/selected states, and key data highlights.
- **Semantic Color - Green (3%)**: `var(--success)`. Used ONLY for positive trends (e.g., up arrows), success states, and checkmarks. NEVER use green for primary buttons or general navigation.

## 2. Hard Color Constraints (Anti-Defaults)
- **NO Pure Black/White**: NEVER use `#000000` or `#ffffff`. 
  - Light mode bg: `#fafafa` or `#f8fafc`. Text: `#111111` or `#0f172a`.
  - Dark mode bg: `#0f0f0f` or `#020617`. Text: `#f0f0f0` or `#f8fafc`.
- **NO Decorative Gradients**: NEVER use two-stop gradients (e.g., Blue to Green) for backgrounds or empty spaces. Use flat surfaces with strong typography instead.
- **Translucent Borders**: NEVER use solid gray for card borders or dividers. ALWAYS use semi-transparent colors (e.g., `rgba(0,0,0,0.08)` for light mode, `rgba(255,255,255,0.08)` for dark mode).

## 3. Accent Discipline (Max 2 Rule)
- **Hard Cap**: Maximum 2 visible uses of the Primary Accent (Blue) background fill per screen (e.g., 1 Primary Button + 1 Badge).
- **Link Downgrade**: If a screen has a Primary CTA, downgrade all text links to `var(--fg)` with an underline. Do not flood the screen with blue text.
- **Focus Rings**: Limit focus rings to 2px solid with 50% opacity of the accent color.

## 4. Contrast Minimums (Strict Gates)
- Body text (≤16px) on any background MUST have a contrast ratio of >= **4.5:1**.
- If the Primary Blue fails the 4.5:1 text contrast test on a light background, you MUST use a darker shade (e.g., `600` or `700` level) for text/icons, reserving the base blue ONLY for large background fills.
- Green text on light backgrounds MUST be darkened to ensure readability.

## 5. Execution
When writing CSS/Tailwind:
- DO NOT invent new colors outside of Neutrals, Blue, and Green.
- Prioritize semantic names (e.g., `bg-blue-600`, `text-emerald-600`) and apply them functionally, not decoratively.
</design_rules>