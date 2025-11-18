import React, { useState } from 'react';
import { View, Text, TextInput, Button, Alert } from 'react-native';
import { gql } from '@apollo/client';
import { useMutation } from '@apollo/client/react';
import { setItem } from '../../utils/storage';
import { router } from 'expo-router';
import { ThemedText } from '@/components/themed-text';

const REGISTER_MUTATION = gql`
  mutation Register($email: String!, $name: String!, $password: String!) {
    register(email: $email, name: $name, password: $password) { token user { id email name } }
  }
`;

export default function RegisterScreen() {
  const [email, setEmail] = useState('');
  const [name, setName] = useState('');
  const [password, setPassword] = useState('');
  const [registerMutation, { loading }] = useMutation(REGISTER_MUTATION);

  const onSubmit = async () => {
    // Basic client-side validation
    if (!email?.trim() || !name?.trim() || !password) {
      Alert.alert('Validation', 'Please enter email, name and password');
      return;
    }

    try {
      const { data } = await registerMutation({ variables: { email, name, password } });
      const token = (data as any)?.register?.token as string | undefined;
      if (token) {
        await setItem('jwt', token);
        router.replace('/(tabs)');
      } else {
        Alert.alert('Registration failed', 'No token returned');
      }
    } catch (e: any) {
      Alert.alert('Registration error', e.message ?? String(e));
    }
  };

  return (
    <View style={{ padding: 16, gap: 12 }}>
      <ThemedText>Email</ThemedText>
      <TextInput value={email} onChangeText={setEmail} autoCapitalize="none" keyboardType="email-address" style={{ borderWidth: 1, padding: 8 }} />
      <ThemedText>Name</ThemedText>
      <TextInput value={name} onChangeText={setName} style={{ borderWidth: 1, padding: 8 }} />
      <ThemedText>Password</ThemedText>
      <TextInput value={password} onChangeText={setPassword} secureTextEntry style={{ borderWidth: 1, padding: 8 }} />
      <Button title={loading ? 'Registering…' : 'Register'} onPress={onSubmit} disabled={loading} />
      <View style={{ height: 12 }} />
      <Button title="Go to Login" onPress={() => router.back()} />
    </View>
  );
}
