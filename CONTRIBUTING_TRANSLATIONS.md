# Contributing Translations

Thank you for helping make WoL-Web accessible worldwide!

## Quick Start (3 Steps)

### 1. Create Your Translation File

Copy `apps/web/src/lib/i18n/en.json` and save it with your ISO 639-1 language code:

- Spanish: `es.json`
- French: `fr.json`
- German: `de.json`
- Ukrainian: `uk.json`
- etc.

**Only translate the values, keep all keys in English.**

### 2. Register Your Language

Edit `apps/web/src/lib/stores/locale.ts`:

```typescript
// 1. Import your file
import es from '$lib/i18n/es.json';

// 2. Add to AVAILABLE_LANGUAGES array
{ code: 'es', name: 'Spanish', nativeName: 'Español' }

// 3. Add to translations object
const translations = {
  en,
  uk,
  es  // <-- add here
};
```

### 3. Test and Submit

```bash
cd apps/web
bun install
bun run dev
```

Open the app, select your language from the footer, and verify all pages work correctly.

Then create a Pull Request with your translation!

## Translation Guidelines

### Structure

The translation file has two sections:

- **`ui`** - Interface text (buttons, labels, forms)
- **`messages`** - Notifications and errors (toasts, validation)

### Important Rules

1. **Keep all JSON keys in English** - only translate the values
2. **Preserve placeholders** like `{count}` or `{name}` exactly as-is
3. **Keep code examples unchanged**:
   - IP addresses: `192.168.1.100`
   - MAC addresses: `AA:BB:CC:DD:EE:FF`
   - Config keys: `enable_per_host_interfaces=true`
4. **Keep technical terms** like "MAC", "IP", "WoL", "ARP" (or use standard translations)
5. **Match the English structure** - same number of keys, same nesting

### Tips

- Use the actual app to see where text appears
- Keep button text short (UI space is limited)
- Be consistent with terminology throughout
- Test on both desktop and mobile
- Have a native speaker review if possible

## Example

From `en.json`:

```json
{
  "ui": {
    "common": {
      "add": "Add",
      "cancel": "Cancel"
    }
  },
  "messages": {
    "host": {
      "createSuccess": "Host created"
    }
  }
}
```

Your translation (`es.json`):

```json
{
  "ui": {
    "common": {
      "add": "Añadir",
      "cancel": "Cancelar"
    }
  },
  "messages": {
    "host": {
      "createSuccess": "Host creado"
    }
  }
}
```

## Need Help?

- Open an issue with the `translation` label
- Check existing translations (`en.json`, `uk.json`) as examples
- Ask questions in your Pull Request

## Supported Languages

- English (`en`)
- Ukrainian (`uk`)

**Your language here!**

---

Thank you for contributing!
