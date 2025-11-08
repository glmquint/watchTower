import { DarkTheme, DefaultTheme, ThemeProvider } from '@react-navigation/native';
import { Stack, useSegments } from 'expo-router';
import { StatusBar } from 'expo-status-bar';
import 'react-native-reanimated';
import { SafeAreaProvider } from 'react-native-safe-area-context';

import { useColorScheme } from '@/hooks/use-color-scheme';
import { ApolloProvider } from '@/components/ApolloProvider';
import * as SecureStore from 'expo-secure-store';
import { useEffect } from 'react';
import { router, usePathname } from 'expo-router';

export const unstable_settings = {
  anchor: '(tabs)',
};

export default function RootLayout() {
  const colorScheme = useColorScheme();
  const pathname = usePathname();
  const segments = useSegments();

  useEffect(() => {
    (async () => {
      const token = await SecureStore.getItemAsync('jwt');
      // Determine if we're on an auth route. Expo Router may normalize pathnames differently
      // across platforms, so we fall back to segment inspection plus pathname substring.
      // Consider several forms of the auth route: grouped '(auth)' segments, or
      // plain '/login' or '/register' pathnames that Expo Router may normalize to.
      const isAuthRoute =
        segments[0] === '(auth)' ||
        pathname?.includes('/(auth)/') ||
        pathname?.includes('/login') ||
        pathname?.includes('/register');
      // Debug trace to help diagnose unexpected redirects.
      console.log('[AuthGate] pathname=', pathname, 'segments=', segments, 'token?=', !!token, 'isAuthRoute=', isAuthRoute);
      if (!token && !isAuthRoute) {
        router.replace('/(auth)/login');
        return;
      }
      if (token && isAuthRoute) {
        router.replace('/(tabs)');
        return;
      }
    })();
  }, [pathname]);

  return (
    <ApolloProvider>
      <SafeAreaProvider>
        <ThemeProvider value={colorScheme === 'dark' ? DarkTheme : DefaultTheme}>
          <Stack>
            <Stack.Screen name="(auth)/login" options={{ title: 'Login' }} />
            <Stack.Screen name="(auth)/register" options={{ title: 'Register' }} />
            <Stack.Screen name="(tabs)" options={{ headerShown: false }} />
            <Stack.Screen name="modal" options={{ presentation: 'modal', title: 'Modal' }} />
          </Stack>
          <StatusBar style="auto" />
        </ThemeProvider>
      </SafeAreaProvider>
    </ApolloProvider>
  );
}
