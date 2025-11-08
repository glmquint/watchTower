import React from 'react';
import { ApolloClient, InMemoryCache, HttpLink, split } from '@apollo/client';
import { ApolloProvider as Provider } from '@apollo/client/react';
import Constants from 'expo-constants';
import * as SecureStore from 'expo-secure-store';
import { GraphQLWsLink } from '@apollo/client/link/subscriptions';
import { createClient } from 'graphql-ws';
import { setContext } from '@apollo/client/link/context';
import { getMainDefinition } from '@apollo/client/utilities';

const API_URL = (Constants?.expoConfig?.extra as any)?.apiUrl || process.env.EXPO_PUBLIC_API_URL || 'http://localhost:8080/query';
const WS_URL = API_URL.replace(/^http/, 'ws');

const httpLink = new HttpLink({ uri: API_URL });

const authLink = setContext(async (_, { headers }) => {
  const token = await SecureStore.getItemAsync('jwt');
  return {
    headers: {
      ...headers,
      Authorization: token ? `Bearer ${token}` : undefined,
    },
  };
});

const wsLink = new GraphQLWsLink(createClient({
  url: WS_URL,
  connectionParams: async () => {
    const token = await SecureStore.getItemAsync('jwt');
    return token ? { Authorization: `Bearer ${token}` } : {};
  },
}));

const splitLink = split(
  ({ query }) => {
    const def = getMainDefinition(query);
    return def.kind === 'OperationDefinition' && def.operation === 'subscription';
  },
  wsLink,
  authLink.concat(httpLink)
);

const client = new ApolloClient({
  cache: new InMemoryCache(),
  link: splitLink,
});

export const ApolloProvider: React.FC<React.PropsWithChildren> = ({ children }) => {
  return <Provider client={client}>{children}</Provider>;
};
