import React from 'react';
import { ApolloClient, InMemoryCache, HttpLink } from '@apollo/client';
import { ApolloProvider as Provider } from '@apollo/client/react';
import Constants from 'expo-constants';

const API_URL = (Constants?.expoConfig?.extra as any)?.apiUrl || process.env.EXPO_PUBLIC_API_URL || 'http://localhost:8080/query';

const client = new ApolloClient({
  cache: new InMemoryCache(),
  link: new HttpLink({ uri: API_URL }),
});

export const ApolloProvider: React.FC<React.PropsWithChildren> = ({ children }) => {
  return <Provider client={client}>{children}</Provider>;
};
