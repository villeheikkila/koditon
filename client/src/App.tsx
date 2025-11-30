import { useEffect, useMemo, useState } from "react";
import "./App.css";
import type { listCitiesResponse } from "./generated/default/default.ts";
import { listCities } from "./generated/default/default.ts";
import type { City, Neighborhood } from "./generated/models";

type LoadState =
  | { state: "idle" | "ready" }
  | { state: "loading"; message: string }
  | { state: "error"; message: string };

type ListCitiesSuccess = Extract<listCitiesResponse, { status: 200 }>;

const isListCitiesSuccess = (
  response: listCitiesResponse,
): response is ListCitiesSuccess => response.status === 200;

const normalizeCities = (payload: City[] | null | undefined): City[] =>
  Array.isArray(payload)
    ? payload.filter((city): city is City => Boolean(city))
    : [];

function App() {
  const [cities, setCities] = useState<City[]>([]);
  const [selectedCityId, setSelectedCityId] = useState<City["id"] | null>(null);
  const [selectedNeighborhoodId, setSelectedNeighborhoodId] = useState<
    Neighborhood["id"] | null
  >(null);
  const [status, setStatus] = useState<LoadState>({ state: "idle" });

  useEffect(() => {
    const controller = new AbortController();

    const loadCities = async () => {
      setStatus({ state: "loading", message: "Loading cities..." });

      try {
        const response = await listCities({ signal: controller.signal });

        if (!isListCitiesSuccess(response)) {
          setStatus({
            state: "error",
            message: "Failed to load cities",
          });
          return;
        }

        const payload = normalizeCities(response.data);
        setCities(payload);

        if (payload.length > 0) {
          setSelectedCityId(payload[0]?.id ?? null);
          const firstNeighborhood = payload[0]?.neighborhoods?.[0];
          setSelectedNeighborhoodId(firstNeighborhood?.id ?? null);
        } else {
          setSelectedCityId(null);
          setSelectedNeighborhoodId(null);
        }

        setStatus({ state: "ready" });
      } catch (error) {
        if (controller.signal.aborted) {
          return;
        }
        const fallbackMessage =
          error instanceof Error ? error.message : "Unknown error";
        setStatus({ state: "error", message: fallbackMessage });
      }
    };

    loadCities();

    return () => controller.abort();
  }, []);

  useEffect(() => {
    if (!selectedCityId) {
      setSelectedNeighborhoodId(null);
      return;
    }

    const city = cities.find((item) => item.id === selectedCityId);
    const neighborhoods = city?.neighborhoods ?? [];

    if (neighborhoods.length === 0) {
      setSelectedNeighborhoodId(null);
      return;
    }

    const neighborhoodStillSelected = neighborhoods.some(
      (neighborhood) => neighborhood.id === selectedNeighborhoodId,
    );

    if (!neighborhoodStillSelected) {
      setSelectedNeighborhoodId(neighborhoods[0]?.id ?? null);
    }
  }, [cities, selectedCityId, selectedNeighborhoodId]);

  const selectedCity = useMemo(
    () =>
      selectedCityId ? cities.find((city) => city.id === selectedCityId) : null,
    [cities, selectedCityId],
  );

  const neighborhoods = useMemo<Neighborhood[]>(
    () => selectedCity?.neighborhoods ?? [],
    [selectedCity],
  );

  return (
    <main className="app">
      <header className="hero">
        <p className="eyebrow">Koditon - Cities</p>
        <h1>Choose a city and neighborhood</h1>
        <p className="lede">
          Pick a city to see its neighborhoods. Data is fetched from the Koditon
          API using the @generated client.
        </p>
      </header>

      <section className="panel">
        <div className="selectors">
          <div className="field">
            <label htmlFor="city-select">City</label>
            <select
              id="city-select"
              value={selectedCityId ?? ""}
              onChange={(event) =>
                setSelectedCityId(event.target.value || null)
              }
              disabled={status.state === "loading" || cities.length === 0}
            >
              {status.state === "loading" && <option>Loading...</option>}
              {status.state !== "loading" && cities.length === 0 && (
                <option>No cities available</option>
              )}
              {cities.map((city) => (
                <option key={city.id} value={city.id}>
                  {city.name}
                </option>
              ))}
            </select>
          </div>

          <div className="field">
            <label htmlFor="neighborhood-select">Neighborhood</label>
            <select
              id="neighborhood-select"
              value={selectedNeighborhoodId ?? ""}
              onChange={(event) =>
                setSelectedNeighborhoodId(event.target.value || null)
              }
              disabled={
                neighborhoods.length === 0 ||
                status.state !== "ready" ||
                !selectedCityId
              }
            >
              {neighborhoods.length === 0 && <option>No neighborhoods</option>}
              {neighborhoods.map((neighborhood) => (
                <option key={neighborhood.id} value={neighborhood.id}>
                  {neighborhood.name} ({neighborhood.postal_code})
                </option>
              ))}
            </select>
          </div>
        </div>

        <div className="status">
          {status.state === "loading" && <p>{status.message}</p>}
          {status.state === "error" && (
            <p className="error">Error: {status.message || "Unknown error"}</p>
          )}
          {status.state === "ready" && selectedCity && (
            <p>
              Selected {selectedCity.name}. Neighborhoods:{" "}
              {neighborhoods.length || "none listed"}.
            </p>
          )}
        </div>
      </section>
    </main>
  );
}

export default App;
