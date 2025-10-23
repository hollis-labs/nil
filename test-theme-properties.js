// Quick test to verify theme properties
const { themePresets } = require('./frontend/src/theme/theme.ts');

console.log('Default theme keys:', Object.keys(themePresets.default));
console.log('Default theme scrollbar properties:');
console.log('  scrollbarBg:', themePresets.default.scrollbarBg);
console.log('  scrollbarThumb:', themePresets.default.scrollbarThumb);
console.log('  scrollbarThumbHover:', themePresets.default.scrollbarThumbHover);
