/**
 * Metro configuration for React Native / Expo.
 * Ensure Metro knows to handle .ts/.tsx files (including in node_modules packages)
 * so packages that ship TypeScript source (like some expo runtime helpers) are
 * transpiled by Babel.
 */
const { getDefaultConfig } = require('expo/metro-config');

const config = getDefaultConfig(__dirname);

// Ensure TypeScript extensions are supported
config.resolver.sourceExts = config.resolver.sourceExts || [];
['ts', 'tsx', 'cjs'].forEach((ext) => {
  if (!config.resolver.sourceExts.includes(ext)) config.resolver.sourceExts.push(ext);
});

module.exports = config;
