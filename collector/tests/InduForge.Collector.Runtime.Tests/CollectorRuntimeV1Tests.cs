namespace InduForge.Collector.Runtime.Tests;

using System.Text.Json;

public sealed class CollectorRuntimeV1Tests
{
    [Fact]
    public void ValueNormalizerRejectsLossyIntegerAndUnspecifiedDateTime()
    {
        Assert.Throws<InvalidCastException>(() => CollectorValueNormalizer.Normalize(1.5d, "int32"));
        Assert.Throws<InvalidCastException>(() => CollectorValueNormalizer.Normalize(DateTime.SpecifyKind(DateTime.UnixEpoch, DateTimeKind.Unspecified), "datetime"));
    }

    [Fact]
    public void ProgramOptionsAcceptOnlyReferencesAndPaths()
    {
        var options = Program.ParseOptions(["--artifact", "/trusted/artifact.json", "--binding", "/trusted/binding.json", "--index", "/trusted/index.json", "--wal", "/trusted/wal", "--listen", "http://127.0.0.1:18081/", "--production", "true"]);
        Assert.True(options.Production);
        Assert.Throws<CollectorRuntimeConfigurationException>(() => Program.ParseOptions(["--unknown", "x", "--binding", "x", "--index", "x", "--wal", "x", "--listen", "x", "--production", "false"]));
    }

    [Fact]
    public void NormalizerRequiresExactModbusArrayShape()
    {
        var normalized = CollectorValueNormalizer.Normalize(new ushort[] { 1, 2 }, "uint16", 2);
        Assert.Equal("[1,2]", normalized.GetRawText());
        Assert.Throws<InvalidCastException>(() => CollectorValueNormalizer.Normalize(new ushort[] { 1 }, "uint16", 2));
    }

    [Fact]
    public void NatsCredentialIsStrictAndProductionRejectsNoAuth()
    {
        using var token = JsonDocument.Parse("{\"schemaVersion\":\"nats-credential.v1\",\"authType\":\"token\",\"token\":\"secret\"}");
        Assert.NotNull(NatsCredential.Parse(token.RootElement, production: true).ToOptions());
        using var none = JsonDocument.Parse("{\"schemaVersion\":\"nats-credential.v1\",\"authType\":\"none\"}");
        Assert.Throws<CollectorRuntimeConfigurationException>(() => NatsCredential.Parse(none.RootElement, production: true));
        using var resource = JsonDocument.Parse("{\"url\":\"nats://127.0.0.1:4222\"}");
        Assert.NotNull(RetryingNatsPublisher.ParseServerResource(resource.RootElement));
    }

    [Fact]
    public async Task RawReservationUsesConservativeWireAndAckBudget()
    {
        var directory = Path.Combine(Path.GetTempPath(), "induforge-reservation-" + Guid.NewGuid().ToString("N"));
        try
        {
            await using var wal = await DurableWal.OpenAsync(new DurableWalOptions(directory, 1_000_000, 500_000));
            using var reservation = await wal.ReserveRawBatchAsync(2);
            Assert.NotNull(reservation);
            Assert.Equal(2, reservation.Count);
            Assert.Equal(65_929, DurableWal.MaximumRawReservationBytes);
        }
        finally
        {
            if (Directory.Exists(directory)) Directory.Delete(directory, recursive: true);
        }
    }
}
