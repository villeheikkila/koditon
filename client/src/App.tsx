import { useEffect, useMemo, useState } from "react";
import "./App.css";
import { listCities } from "./generated/default/default.ts";
import type { City, Neighborhood } from "./generated/models";

type AsyncState = "idle" | "loading" | "ready" | "error";

function App() {
  const [cities, setCities] = useState<City[]>([]);
  const [selectedCityId, setSelectedCityId] = useState<string>("");
  const [selectedNeighborhoodId, setSelectedNeighborhoodId] =
    useState<string>("");
  const [status, setStatus] = useState<AsyncState>("idle");
  const [statusMessage, setStatusMessage] = useState<string>("");

  useEffect(() => {
    const loadCities = async () => {
      setStatus("loading");
      setStatusMessage("Loading cities...");

      try {
        const response = await listCities();

        if (response.status !== 200) {
          setStatus("error");
          setStatusMessage("Failed to load cities");
          return;
        }

        const payload = response.data ?? [];
        setCities(payload);

        if (payload.length > 0) {
          setSelectedCityId(payload[0].id);
          const firstNeighborhood = payload[0].neighborhoods?.[0];
          setSelectedNeighborhoodId(firstNeighborhood?.id ?? "");
        }

        setStatus("ready");
        setStatusMessage("");
      } catch (error) {
        setStatus("error");
        const fallbackMessage =
          error instanceof Error ? error.message : "Unknown error";
        setStatusMessage(fallbackMessage);
      }
    };

    loadCities();
  }, []);

  useEffect(() => {
    if (!selectedCityId) {
      return;
    }

    const city = cities.find((item) => item.id === selectedCityId);
    const neighborhoods = city?.neighborhoods ?? [];

    if (neighborhoods.length === 0) {
      setSelectedNeighborhoodId("");
      return;
    }

    const neighborhoodStillSelected = neighborhoods.some(
      (neighborhood) => neighborhood.id === selectedNeighborhoodId,
    );

    if (!neighborhoodStillSelected) {
      setSelectedNeighborhoodId(neighborhoods[0].id);
    }
  }, [cities, selectedCityId, selectedNeighborhoodId]);

  const selectedCity = useMemo(
    () => cities.find((city) => city.id === selectedCityId),
    [cities, selectedCityId],
  );

  const neighborhoods: Neighborhood[] = useMemo(
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
              value={selectedCityId}
              onChange={(event) => setSelectedCityId(event.target.value)}
              disabled={status === "loading" || cities.length === 0}
            >
              {status === "loading" && <option>Loading...</option>}
              {status !== "loading" && cities.length === 0 && (
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
              value={selectedNeighborhoodId}
              onChange={(event) =>
                setSelectedNeighborhoodId(event.target.value)
              }
              disabled={neighborhoods.length === 0 || status !== "ready"}
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
          {status === "loading" && <p>Loading cities...</p>}
          {status === "error" && (
            <p className="error">Error: {statusMessage || "Unknown error"}</p>
          )}
          {status === "ready" && selectedCity && (
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
