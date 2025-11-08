import React, { useMemo, useState, useCallback } from 'react';
import { FlatList, Text, View, ActivityIndicator, Pressable } from 'react-native';
import { SafeAreaView } from 'react-native-safe-area-context';
import { gql } from '@apollo/client';
import { useQuery } from '@apollo/client/react';
import { useSubscription } from '@apollo/client/react';
import { ThemedText } from '@/components/themed-text';
import { Link } from 'expo-router';
import { useFocusEffect } from '@react-navigation/native';

export const INCIDENTS_QUERY = gql`
  query Incidents($status: String) {
    incidents(status: $status) { id title severity status details }
  }
`;

const INCIDENTS_SUB = gql`
  subscription OnNewIncident {
    onNewIncident { id title }
  }
`;

export default function IncidentsScreen() {
  const [filter, setFilter] = useState<'all' | 'open' | 'acknowledged'>('all');
  const variables = useMemo(() => ({ status: filter === 'all' ? null : filter }), [filter]);
  const { data, loading, error, client, refetch } = useQuery<{ incidents: { id: string; title: string; severity: string; status: string; details?: string | null; }[] }>(
    INCIDENTS_QUERY,
    {
      variables,
      fetchPolicy: 'cache-and-network',
      nextFetchPolicy: 'cache-first',
      notifyOnNetworkStatusChange: true,
    }
  );

  // When the screen regains focus (e.g., after acknowledging on detail), refresh the list
  useFocusEffect(
    useCallback(() => {
      refetch();
      return () => {};
    }, [refetch, variables.status])
  );
  useSubscription(INCIDENTS_SUB, {
    onData: ({ client, data }) => {
      const payload: any = data.data;
      const newIncident = payload?.onNewIncident;
      if (!newIncident) return;
      try {
        const existing = client.readQuery<{ incidents: any[] }>({ query: INCIDENTS_QUERY, variables })?.incidents ?? [];
        client.writeQuery({
          query: INCIDENTS_QUERY,
          variables,
          data: { incidents: [newIncident, ...existing] },
        });
        // Refetch to hydrate missing fields (severity/status/details) from server
        client.refetchQueries({ include: [INCIDENTS_QUERY] });
      } catch {}
    },
  });

  if (loading) return <ActivityIndicator style={{ marginTop: 40 }} />;
  if (error) return <Text style={{ color: 'red' }}>Error: {error.message}</Text>;

  return (
    <SafeAreaView style={{ flex: 1 }}>
      {/* Simple status filter "tabs" */}
      <View style={{ flexDirection: 'row', paddingHorizontal: 12, paddingTop: 8, gap: 8 }}>
        {(['all', 'open', 'acknowledged'] as const).map((key) => (
          <Pressable
            key={key}
            onPress={() => setFilter(key)}
            style={{
              paddingVertical: 6,
              paddingHorizontal: 12,
              borderRadius: 999,
              backgroundColor: filter === key ? '#111827' : '#e5e7eb',
            }}
          >
            <Text style={{ color: filter === key ? 'white' : '#111827', fontWeight: '600', textTransform: 'capitalize' }}>{key}</Text>
          </Pressable>
        ))}
      </View>
      <FlatList
        data={data?.incidents ?? []}
        keyExtractor={(item) => item.id}
        renderItem={({ item }) => {
          return (
            <Link href={{ pathname: '/(tabs)/incidents/[id]', params: { id: item.id } }} asChild>
              <Pressable style={{ padding: 12, borderBottomWidth: 1, borderColor: '#eee' }}>
                <ThemedText style={{ fontSize: 16, fontWeight: '600' }}>{item.title}</ThemedText>
                <View style={{ flexDirection: 'row', gap: 8, marginTop: 6 }}>
                  <Badge label={item.severity} color={severityColor(item.severity)} />
                  <Badge label={item.status} color={item.status === 'acknowledged' ? '#059669' : '#f59e0b'} />
                </View>
                {item.details ? (
                  <ThemedText style={{ marginTop: 6, fontSize: 12 }} numberOfLines={2}>
                    {item.details}
                  </ThemedText>
                ) : null}
              </Pressable>
            </Link>
          );
        }}
        ListEmptyComponent={<Text style={{ padding: 20 }}>No incidents.</Text>}
      />
    </SafeAreaView>
  );
}

function Badge({ label, color }: { label: string; color: string }) {
  return (
    <View style={{ backgroundColor: color, paddingHorizontal: 8, paddingVertical: 2, borderRadius: 6 }}>
      <Text style={{ color: 'white', fontSize: 12, fontWeight: '600', textTransform: 'capitalize' }}>{label}</Text>
    </View>
  );
}

function severityColor(sev: string) {
  switch (sev?.toLowerCase()) {
    case 'high':
    case 'critical':
      return '#dc2626';
    case 'low':
      return '#2563eb';
    default:
      return '#f59e0b'; // medium/default
  }
}
