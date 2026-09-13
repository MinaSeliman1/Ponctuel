import { fetchErrorSummary } from '../../src/api/client'

describe('fetchErrorSummary', () => {
  afterEach(() => {
    vi.restoreAllMocks()
  })

  it('requests and returns bounded prediction quality summaries', async () => {
    const fetchMock = vi.spyOn(globalThis, 'fetch').mockResolvedValue(
      new Response(
        JSON.stringify({
          data: {
            errorSummary: [
              {
                routeId: '51',
                horizonSeconds: 300,
                sampleCount: 4,
                meanErrorSeconds: 42.5,
                onTimeRate: 0.75,
              },
              {
                routeId: null,
                horizonSeconds: 600,
                sampleCount: 2,
                meanErrorSeconds: -15,
                onTimeRate: 1,
              },
            ],
          },
        }),
        { status: 200, headers: { 'Content-Type': 'application/json' } },
      ),
    )

    await expect(fetchErrorSummary(250)).resolves.toEqual([
      {
        routeId: '51',
        horizonSeconds: 300,
        sampleCount: 4,
        meanErrorSeconds: 42.5,
        onTimeRate: 0.75,
      },
      {
        routeId: null,
        horizonSeconds: 600,
        sampleCount: 2,
        meanErrorSeconds: -15,
        onTimeRate: 1,
      },
    ])

    expect(fetchMock).toHaveBeenCalledTimes(1)
    const request = JSON.parse(String(fetchMock.mock.calls[0]?.[1]?.body)) as {
      query: string
      variables: { limit: number }
    }
    expect(request.variables).toEqual({ limit: 100 })
    expect(request.query).toContain('errorSummary(limit: $limit)')
    expect(request.query).toContain('meanErrorSeconds')
  })

  it('rejects an unavailable API response', async () => {
    vi.spyOn(globalThis, 'fetch').mockResolvedValue(new Response('unavailable', { status: 503 }))

    await expect(fetchErrorSummary()).rejects.toThrow('api-unavailable')
  })
})
