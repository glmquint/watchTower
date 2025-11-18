import React, { useState, useEffect, useRef } from 'react';
import { View, Text, TextInput, Button, Alert } from 'react-native';
import { gql } from '@apollo/client';
import { useMutation } from '@apollo/client/react';
import { getItem, deleteItem, setItem } from '../../utils/storage';
import { router } from 'expo-router';
import { ThemedText } from '@/components/themed-text';

const LOGIN_MUTATION = gql`
  mutation Login($email: String!, $password: String!) {
    login(email: $email, password: $password) { token user { id email name } }
  }
`;

export default function LoginScreen() {
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [login, { loading }] = useMutation(LOGIN_MUTATION);
  const promptedRef = useRef(false);

  useEffect(() => {
    (async () => {
      // If a token is already present, prompt the user to either continue with it
      // or clear it so they can register/login with different credentials.
      try {
        const token = await getItem('jwt');
        if (token && !promptedRef.current) {
          promptedRef.current = true;
          // show a simplified choice: continue with the current session or use another account
          Alert.alert(
            'Already signed in',
            'We found an active session. Continue signed in, or sign in with another account?',
            [
              { text: 'Continue', onPress: () => router.replace('/') },
              {
                text: 'Use another account',
                onPress: async () => {
                  await deleteItem('jwt');
                },
              },
            ],
            { cancelable: false }
          );
        }
      } catch (e) {
        // ignore
      }
    })();
  }, []);

  const onSubmit = async () => {
    // Basic client-side validation
    if (!email?.trim() || !password) {
      Alert.alert('Validation', 'Please enter both email and password');
      return;
    }
    try {
      const { data } = await login({ variables: { email, password } });
      const token = (data as any)?.login?.token as string | undefined;
      if (token) {
        await setItem('jwt', token);
        router.replace('/');
      } else {
        Alert.alert('Login failed', 'No token returned');
      }
    } catch (e: any) {
      Alert.alert('Login error', e.message ?? String(e));
    }
  };

  return (
    <View style={{ padding: 16, gap: 12 }}>
      <ThemedText>Email</ThemedText>
      <TextInput value={email} onChangeText={setEmail} autoCapitalize="none" keyboardType="email-address" style={{ borderWidth: 1, padding: 8 }} />
      <ThemedText>Password</ThemedText>
      <TextInput value={password} onChangeText={setPassword} secureTextEntry style={{ borderWidth: 1, padding: 8 }} />
      <Button title={loading ? 'Logging in…' : 'Login'} onPress={onSubmit} disabled={loading} />
      <View style={{ height: 12 }} />
      <Button title="Go to Register" onPress={() => router.push('/register')} />
    </View>
  );
}
