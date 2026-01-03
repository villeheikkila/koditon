import { ThemedText } from "@/components/themed-text";
import { ThemedView } from "@/components/themed-view";
import { pricesTransactions } from "@/generated/default/default";
import type { PostalCode, PricesTransaction } from "@/generated/models";
import { useAppState } from "@/state/use-app-state";
import { useEffect, useMemo, useState } from "react";
import { Pressable, ScrollView, StyleSheet, View } from "react-native";

const emptyPostalCodes: PostalCode[] = [];

export default function HomeScreen() {
  const { cities, loadingCities, error: citiesError } = useAppState();
  const [selectedCityId, setSelectedCityId] = useState<string | null>(null);
  const [selectedPostalCodeId, setSelectedPostalCodeId] = useState<
    string | null
  >(null);
  const [transactions, setTransactions] = useState<PricesTransaction[]>([]);
  const [loadingTransactions, setLoadingTransactions] = useState(false);
  const [transactionsError, setTransactionsError] = useState<string | null>(
    null,
  );
  const selectedCity = useMemo(
    () => cities.find((city) => city.id === selectedCityId) ?? null,
    [cities, selectedCityId],
  );
  const postalCodes = selectedCity?.postal_codes ?? emptyPostalCodes;
  useEffect(() => {
    if (cities.length === 0) {
      setSelectedCityId((current) => (current === null ? current : null));
      return;
    }
    setSelectedCityId((current) => current ?? cities[0].id);
  }, [cities]);
  useEffect(() => {
    if (selectedCityId && postalCodes.length > 0) {
      if (
        !selectedPostalCodeId ||
        !postalCodes.some(
          (postalCode) => postalCode.id === selectedPostalCodeId,
        )
      ) {
        setSelectedPostalCodeId(postalCodes[0].id);
      }
    } else {
      setSelectedPostalCodeId((current) => (current === null ? current : null));
      setTransactions((current) => (current.length === 0 ? current : []));
      setTransactionsError((current) => (current === null ? current : null));
    }
  }, [selectedCityId, postalCodes, selectedPostalCodeId]);
  useEffect(() => {
    const loadTransactions = async () => {
      if (!selectedCityId || !selectedPostalCodeId) {
        return;
      }
      setLoadingTransactions(true);
      setTransactionsError(null);
      try {
        const response = await pricesTransactions({
          municipality_id: selectedCityId,
          postal_code_id: selectedPostalCodeId,
        });
        if (response.status !== 200) {
          throw new Error("Failed to load transactions");
        }
        const nextTransactions = response.data.transactions ?? [];
        setTransactions(nextTransactions);
      } catch {
        setTransactionsError("Failed to load transactions");
        setTransactions([]);
      } finally {
        setLoadingTransactions(false);
      }
    };
    void loadTransactions();
  }, [selectedCityId, selectedPostalCodeId]);
  return (
    <ScrollView contentContainerStyle={styles.container}>
      <ThemedView style={styles.titleContainer}>
        <ThemedText type="title">Prices</ThemedText>
      </ThemedView>
      <ThemedView style={styles.section}>
        <ThemedText type="subtitle">City</ThemedText>
        <View style={styles.picker}>
          {loadingCities ? <ThemedText>Loading cities...</ThemedText> : null}
          {citiesError ? <ThemedText>{citiesError}</ThemedText> : null}
          {cities.map((city) => (
            <Pressable
              key={city.id}
              onPress={() => setSelectedCityId(city.id)}
              style={[
                styles.pill,
                selectedCityId === city.id ? styles.pillActive : null,
              ]}
            >
              <ThemedText type="defaultSemiBold">{city.name_fi}</ThemedText>
            </Pressable>
          ))}
        </View>
      </ThemedView>
      <ThemedView style={styles.section}>
        <ThemedText type="subtitle">Postal Code</ThemedText>
        <View style={styles.picker}>
          {postalCodes.length === 0 && !loadingCities ? (
            <ThemedText>No postal codes available</ThemedText>
          ) : null}
          {postalCodes.map((postalCode) => (
            <Pressable
              key={postalCode.id}
              onPress={() => setSelectedPostalCodeId(postalCode.id)}
              style={[
                styles.pill,
                selectedPostalCodeId === postalCode.id
                  ? styles.pillActive
                  : null,
              ]}
            >
              <ThemedText type="defaultSemiBold">{postalCode.code}</ThemedText>
            </Pressable>
          ))}
        </View>
      </ThemedView>
      <ThemedView style={styles.section}>
        <ThemedText type="subtitle">Transactions</ThemedText>
        {loadingTransactions ? (
          <ThemedText>Loading transactions...</ThemedText>
        ) : null}
        {transactionsError ? (
          <ThemedText>{transactionsError}</ThemedText>
        ) : null}
        {transactions.map((tx) => (
          <View key={tx.id} style={styles.transactionCard}>
            <ThemedText type="defaultSemiBold">{tx.description}</ThemedText>
            <ThemedText>
              {tx.postal_code_code} · {tx.municipality_name_fi}
            </ThemedText>
            <ThemedText>
              {tx.price.toLocaleString("fi-FI")} € · {tx.area.toFixed(1)} m²
            </ThemedText>
          </View>
        ))}
        {transactions.length === 0 &&
        !loadingTransactions &&
        !transactionsError ? (
          <ThemedText>No transactions for the selection</ThemedText>
        ) : null}
      </ThemedView>
    </ScrollView>
  );
}

const styles = StyleSheet.create({
  container: {
    padding: 16,
    gap: 16,
  },
  titleContainer: {
    flexDirection: "row",
    alignItems: "center",
    gap: 8,
  },
  section: {
    gap: 8,
    marginBottom: 16,
  },
  picker: {
    flexDirection: "row",
    flexWrap: "wrap",
    gap: 8,
  },
  pill: {
    borderRadius: 16,
    borderWidth: 1,
    borderColor: "#B9B2A7",
    paddingHorizontal: 12,
    paddingVertical: 6,
  },
  pillActive: {
    backgroundColor: "#E8DFD1",
  },
  transactionCard: {
    borderRadius: 12,
    padding: 12,
    backgroundColor: "#F3EEE5",
    gap: 4,
  },
});
