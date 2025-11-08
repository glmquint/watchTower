import React, { useMemo, useState } from 'react';
import { ActivityIndicator, Button, FlatList, Text, TextInput, View } from 'react-native';
import { SafeAreaView } from 'react-native-safe-area-context';
import { useLocalSearchParams } from 'expo-router';
import { gql } from '@apollo/client';
import { useMutation, useQuery } from '@apollo/client/react';
import { useSubscription } from '@apollo/client/react';
import { ThemedText } from '@/components/themed-text';
import { INCIDENTS_QUERY } from '../incidents';

type CommentT = { id: string; text: string; createdAt: string; author?: { id: string; email: string; name: string } | null };
type IncidentT = { id: string; title: string; severity: string; status: string; details?: string | null; comments?: CommentT[] | null };

const INCIDENT_QUERY = gql`
  query Incident($id: ID!) {
    incident(id: $id) {
      id
      title
      severity
      status
      details
      comments { id text createdAt author { id email name } }
    }
  }
`;

const ACK_MUTATION = gql`
  mutation Ack($id: ID!) {
    acknowledgeIncident(incidentId: $id) { id status }
  }
`;

const ADD_COMMENT_MUTATION = gql`
  mutation AddComment($id: ID!, $text: String!) {
    addComment(incidentId: $id, text: $text) { id text createdAt }
  }
`;

const INCIDENT_UPDATE_SUB = gql`
  subscription OnIncidentUpdate($id: ID!) {
    onIncidentUpdate(incidentId: $id) {
      id
      status
      comments { id text createdAt }
    }
  }
`;

export default function IncidentDetailScreen() {
  const { id } = useLocalSearchParams<{ id: string }>();
  const variables = useMemo(() => ({ id }), [id]);
  const { data, loading, error, client } = useQuery<{ incident: IncidentT }>(INCIDENT_QUERY, { variables });
  const [ack, { loading: ackLoading }] = useMutation<{ acknowledgeIncident: { id: string; status: string } }>(ACK_MUTATION);
  const [addComment, { loading: commentLoading }] = useMutation<{ addComment: CommentT }>(ADD_COMMENT_MUTATION);
  const [text, setText] = useState('');

  useSubscription<{ onIncidentUpdate: { id: string; status?: string | null; comments?: CommentT[] | null } }>(INCIDENT_UPDATE_SUB, {
    variables,
    onData: ({ data }) => {
      const inc = data.data?.onIncidentUpdate as { id: string; status?: string | null; comments?: CommentT[] | null } | undefined;
      if (!inc) return;
      try {
        const existing = client.readQuery<any>({ query: INCIDENT_QUERY, variables });
        if (!existing?.incident) return;
        const next = { ...existing.incident };
        if (inc.status) next.status = inc.status;
        if (Array.isArray(inc.comments) && inc.comments.length > 0) {
          next.comments = [...(next.comments ?? []), ...inc.comments];
        }
        client.writeQuery({ query: INCIDENT_QUERY, variables, data: { incident: next } });
      } catch {}
    },
  });

  if (!id) return <Text style={{ padding: 16, color: 'red' }}>Missing incident id</Text>;
  if (loading) return <ActivityIndicator style={{ marginTop: 40 }} />;
  if (error) return <Text style={{ padding: 16, color: 'red' }}>Error: {error.message}</Text>;

  const incident = data?.incident as IncidentT | undefined;
  if (!incident) return <Text style={{ padding: 16 }}>Incident not found.</Text>;

  const onAck = async () => {
    try {
  const { data: resp } = await ack({ variables: { id } });
  const newStatus = resp?.acknowledgeIncident?.status;
      if (newStatus) {
        client.writeQuery({
          query: INCIDENT_QUERY,
          variables,
          data: { incident: { ...incident, status: newStatus } },
        });
      }
      // Ensure incident lists with any filter get refreshed too
      client.refetchQueries({ include: [INCIDENTS_QUERY] });
    } catch (e) {
      // ignore errors for now; UI will reflect via subscription or re-query
    }
  };

  const onAddComment = async () => {
    if (!text.trim()) return;
    try {
  const { data: resp } = await addComment({ variables: { id, text } });
  const c = resp?.addComment;
      if (c) {
        const existing = client.readQuery<any>({ query: INCIDENT_QUERY, variables });
        client.writeQuery({
          query: INCIDENT_QUERY,
          variables,
          data: { incident: { ...existing.incident, comments: [...(existing.incident.comments ?? []), c] } },
        });
      }
      setText('');
    } catch (e) {
      // ignore for now
    }
  };

  return (
    <SafeAreaView style={{ flex: 1, padding: 12 }}>
      <ThemedText style={{ fontSize: 20, fontWeight: '700' }}>{incident.title}</ThemedText>
      <View style={{ flexDirection: 'row', gap: 8, marginTop: 8 }}>
        <Badge label={incident.severity} color={severityColor(incident.severity)} />
        <Badge label={incident.status} color={incident.status === 'acknowledged' ? '#059669' : '#f59e0b'} />
      </View>
      {incident.details ? (
        <ThemedText style={{ marginTop: 10 }}>{incident.details}</ThemedText>
      ) : null}

      {incident.status !== 'acknowledged' ? (
        <View style={{ marginTop: 12, alignSelf: 'flex-start' }}>
          <Button title={ackLoading ? 'Acknowledging…' : 'Acknowledge'} onPress={onAck} disabled={ackLoading} />
        </View>
      ) : null}

      <ThemedText style={{ fontSize: 16, fontWeight: '600', marginTop: 16 }}>Comments</ThemedText>
      <FlatList
        data={incident.comments ?? []}
        keyExtractor={(item) => item.id}
        renderItem={({ item }) => (
          <View style={{ paddingVertical: 8, borderBottomWidth: 1, borderColor: '#eee' }}>
            <ThemedText>{item.text}</ThemedText>
            <ThemedText style={{ fontSize: 11, color: '#6b7280' }}>{item.createdAt}</ThemedText>
          </View>
        )}
        ListEmptyComponent={<Text style={{ paddingVertical: 8 }}>No comments yet.</Text>}
        style={{ marginTop: 8 }}
      />

      <View style={{ flexDirection: 'row', gap: 8, alignItems: 'center', marginTop: 8 }}>
        <TextInput
          value={text}
          onChangeText={setText}
          placeholder="Add a comment"
          style={{ flex: 1, borderWidth: 1, borderColor: '#e5e7eb', borderRadius: 8, paddingHorizontal: 10, paddingVertical: 8 }}
        />
        <Button title={commentLoading ? 'Sending…' : 'Send'} onPress={onAddComment} disabled={commentLoading || !text.trim()} />
      </View>
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
      return '#f59e0b';
  }
}
