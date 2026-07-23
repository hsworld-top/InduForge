using System.Buffers.Binary;
using System.Net;
using System.Net.Sockets;

namespace InduForge.Collector.Drivers.ModbusTcp.Tests;

internal sealed class ModbusTcpTestServer : IAsyncDisposable
{
    private readonly TcpListener _listener = new(IPAddress.Loopback, 0);
    private readonly CancellationTokenSource _cancellation = new();
    private readonly Task _acceptLoop;

    public ModbusTcpTestServer()
    {
        _listener.Start();
        Port = ((IPEndPoint)_listener.LocalEndpoint).Port;
        _acceptLoop = AcceptLoopAsync(_cancellation.Token);
    }

    public int Port { get; }

    public Dictionary<ushort, bool> Coils { get; } = [];

    public Dictionary<ushort, bool> DiscreteInputs { get; } = [];

    public Dictionary<ushort, ushort> HoldingRegisters { get; } = [];

    public Dictionary<ushort, ushort> InputRegisters { get; } = [];

    public async ValueTask DisposeAsync()
    {
        await _cancellation.CancelAsync();
        _listener.Stop();
        try
        {
            await _acceptLoop.ConfigureAwait(false);
        }
        catch (OperationCanceledException)
        {
        }
        _cancellation.Dispose();
    }

    private async Task AcceptLoopAsync(CancellationToken cancellationToken)
    {
        while (!cancellationToken.IsCancellationRequested)
        {
            TcpClient client;
            try
            {
                client = await _listener.AcceptTcpClientAsync(cancellationToken).ConfigureAwait(false);
            }
            catch (OperationCanceledException)
            {
                break;
            }
            catch (SocketException) when (cancellationToken.IsCancellationRequested)
            {
                break;
            }

            _ = HandleClientAsync(client, cancellationToken);
        }
    }

    private async Task HandleClientAsync(TcpClient client, CancellationToken cancellationToken)
    {
        using (client)
        {
            var stream = client.GetStream();
            var header = new byte[7];
            while (!cancellationToken.IsCancellationRequested)
            {
                try
                {
                    await stream.ReadExactlyAsync(header, cancellationToken).ConfigureAwait(false);
                }
                catch (EndOfStreamException)
                {
                    return;
                }
                catch (OperationCanceledException)
                {
                    return;
                }

                var length = BinaryPrimitives.ReadUInt16BigEndian(header.AsSpan(4, 2));
                if (length < 2) return;
                var pdu = new byte[length - 1];
                await stream.ReadExactlyAsync(pdu, cancellationToken).ConfigureAwait(false);
                var responsePdu = BuildResponse(pdu);
                var responseHeader = new byte[7];
                header.AsSpan(0, 4).CopyTo(responseHeader);
                BinaryPrimitives.WriteUInt16BigEndian(responseHeader.AsSpan(4, 2), checked((ushort)(responsePdu.Length + 1)));
                responseHeader[6] = header[6];
                await stream.WriteAsync(responseHeader, cancellationToken).ConfigureAwait(false);
                await stream.WriteAsync(responsePdu, cancellationToken).ConfigureAwait(false);
            }
        }
    }

    private byte[] BuildResponse(byte[] request)
    {
        if (request.Length < 5) return [0x80, 0x03];
        var function = request[0];
        var startAddress = BinaryPrimitives.ReadUInt16BigEndian(request.AsSpan(1, 2));
        var quantity = BinaryPrimitives.ReadUInt16BigEndian(request.AsSpan(3, 2));
        return function switch
        {
            0x01 => BuildBitResponse(function, startAddress, quantity, Coils),
            0x02 => BuildBitResponse(function, startAddress, quantity, DiscreteInputs),
            0x03 => BuildRegisterResponse(function, startAddress, quantity, HoldingRegisters),
            0x04 => BuildRegisterResponse(function, startAddress, quantity, InputRegisters),
            _ => [(byte)(function | 0x80), 0x01],
        };
    }

    private static byte[] BuildBitResponse(
        byte function,
        ushort startAddress,
        ushort quantity,
        Dictionary<ushort, bool> values)
    {
        var byteCount = checked((byte)((quantity + 7) / 8));
        var response = new byte[2 + byteCount];
        response[0] = function;
        response[1] = byteCount;
        for (var offset = 0; offset < quantity; offset++)
        {
            var address = checked((ushort)(startAddress + offset));
            if (!values.TryGetValue(address, out var value)) return [(byte)(function | 0x80), 0x02];
            if (value) response[2 + offset / 8] |= checked((byte)(1 << (offset % 8)));
        }
        return response;
    }

    private static byte[] BuildRegisterResponse(
        byte function,
        ushort startAddress,
        ushort quantity,
        Dictionary<ushort, ushort> values)
    {
        var response = new byte[2 + quantity * 2];
        response[0] = function;
        response[1] = checked((byte)(quantity * 2));
        for (var offset = 0; offset < quantity; offset++)
        {
            var address = checked((ushort)(startAddress + offset));
            if (!values.TryGetValue(address, out var value)) return [(byte)(function | 0x80), 0x02];
            BinaryPrimitives.WriteUInt16BigEndian(response.AsSpan(2 + offset * 2, 2), value);
        }
        return response;
    }
}
