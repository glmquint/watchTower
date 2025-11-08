import React from 'react';
import { FlatList, Text, View, ActivityIndicator } from 'react-native';
import { SafeAreaView } from 'react-native-safe-area-context';
import { gql} from '@apollo/client';
import { useQuery } from '@apollo/client/react';
import { ThemedText } from '@/components/themed-text';

const INCIDENTS_QUERY = gql`
  query Incidents {
    incidents { id title }
  }
`;

export default function IncidentsScreen() {
  console.log('[IncidentsScreen] render hook');
  const { data, loading, error } = useQuery<{ incidents: { id: string; title: string }[] }>(INCIDENTS_QUERY);

  if (loading) return <ActivityIndicator style={{ marginTop: 40 }} />;
  if (error) return <Text style={{ color: 'red' }}>Error: {error.message}</Text>;

  return (
    <SafeAreaView style={{ flex: 1 }}>
      <FlatList
        data={data?.incidents ?? []}
        keyExtractor={(item) => item.id}
        renderItem={({ item }) => (
          <View style={{ padding: 12, borderBottomWidth: 1, borderColor: '#eee' }}>
            <ThemedText style={{ fontSize: 16, fontWeight: '600' }}>{item.title}</ThemedText>
            <ThemedText style={{ fontSize: 12}}>ID: {item.id}</ThemedText>
          </View>
        )}
        ListEmptyComponent={<Text style={{ padding: 20 }}>No incidents.</Text>}
      />
    </SafeAreaView>
  );
}
