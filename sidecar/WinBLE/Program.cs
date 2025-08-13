using System.Net;
using System.Net.Sockets;
using System.Runtime.InteropServices;
using System.Text;
using System.Text.Json;
using System.IO;
using System.Threading;
using System.Diagnostics;
using System.Collections.Generic;
using Windows.Storage;
using Windows.Devices.Bluetooth;
using Windows.Devices.Bluetooth.Advertisement;
using Windows.Devices.Bluetooth.GenericAttributeProfile;
using Windows.Storage.Streams;
using System.Runtime.InteropServices.WindowsRuntime;

// Lightweight structured logger
void Log(string level, string message, IDictionary<string, object?>? fields = null)
{
    try
    {
        var payload = new Dictionary<string, object?>
        {
            ["ts"] = DateTime.Now.ToString("O"),
            ["level"] = level,
            ["msg"] = message,
            ["pid"] = Environment.ProcessId,
            ["tid"] = Thread.CurrentThread.ManagedThreadId,
        };
        if (fields != null)
        {
            foreach (var kv in fields)
            {
                payload[kv.Key] = kv.Value;
            }
        }
        Console.WriteLine(JsonSerializer.Serialize(payload));
    }
    catch
    {
        Console.WriteLine($"[{DateTime.Now:T}] {level} {message}");
    }
}

// Configure file logging for packaged app (console window may not be visible)
try
{
    var localPath = ApplicationData.Current.LocalFolder.Path;
    var logPath = Path.Combine(localPath, "WinBLE.log");
    Directory.CreateDirectory(Path.GetDirectoryName(logPath)!);
    var logStream = new FileStream(logPath, FileMode.Append, FileAccess.Write, FileShare.Read);
    var logWriter = new StreamWriter(logStream, new UTF8Encoding(false)) { AutoFlush = true };
    Console.SetOut(logWriter);
    Console.SetError(logWriter);
    Log("INFO", "logging initialized", new Dictionary<string, object?> { { "path", logPath } });
}
catch
{
    // If logging setup fails, continue without file logging
}

var server = new TcpListener(IPAddress.Loopback, 8765);
server.Start();
Log("INFO", "WinBLE sidecar listening", new Dictionary<string, object?> { { "endpoint", "127.0.0.1:8765" } });
Log("INFO", "environment", new Dictionary<string, object?>{
    {"os", Environment.OSVersion.ToString()},
    {"arch", RuntimeInformation.ProcessArchitecture.ToString()},
    {"machine", Environment.MachineName},
    {"user", Environment.UserName},
    {"cwd", Environment.CurrentDirectory},
});
try
{
    var pkgLocal = ApplicationData.Current.LocalFolder.Path;
    var pkgTemp = ApplicationData.Current.TemporaryFolder.Path;
    Log("DEBUG", "appdata paths", new Dictionary<string, object?> { { "LocalFolder", pkgLocal }, { "TempFolder", pkgTemp } });
}
catch { }
try
{
    // If packaged, this may succeed
    var pkgId = Windows.ApplicationModel.Package.Current?.Id;
    if (pkgId != null)
    {
        Log("DEBUG", "package id", new Dictionary<string, object?> { { "FullName", pkgId.FullName }, { "FamilyName", pkgId.FamilyName } });
    }
}
catch { }
// Log adapter capabilities at startup for easier diagnostics
try
{
    var startupAdapter = await BluetoothAdapter.GetDefaultAsync();
    if (startupAdapter == null)
    {
        Log("WARN", "No Bluetooth adapter found", null);
    }
    else
    {
        Log("INFO", "Adapter caps", new Dictionary<string, object?>{
            {"LE", startupAdapter.IsLowEnergySupported},
            {"Peripheral", startupAdapter.IsPeripheralRoleSupported},
            {"Central", startupAdapter.IsCentralRoleSupported},
            {"Offload", startupAdapter.IsAdvertisementOffloadSupported},
            {"BTAddr", startupAdapter.BluetoothAddress},
        });
    }
}
catch (Exception ex)
{
    Log("ERROR", "Adapter capability query failed", new Dictionary<string, object?> { { "error", ex.Message } });
}

BluetoothLEAdvertisementPublisher? publisher = null;
GattServiceProvider? serviceProvider = null;
GattLocalCharacteristic? characteristic = null;

// Subscribers for streaming write events back to TCP clients
var subLock = new object();
var subscribers = new HashSet<StreamWriter>();
async Task BroadcastAsync(string json)
{
    List<StreamWriter> writers;
    lock (subLock)
    {
        writers = new List<StreamWriter>(subscribers);
    }
    foreach (var w in writers)
    {
        try { await w.WriteLineAsync(json); } catch { }
    }
}

// (removed duplicate static subscription helpers; using local subLock/subscribers above)

async Task<JsonElement> HandleAsync(JsonElement req)
{
    var action = req.GetProperty("action").GetString();
    Log("INFO", "request", new Dictionary<string, object?> { { "action", action }, { "raw", req.ToString() } });
    switch (action)
    {
        case "advertise_start":
            {
                var p = req.GetProperty("params");
                var svcUuid = p.TryGetProperty("service_uuid", out var v1) ? v1.GetString() : "";
                var localName = p.TryGetProperty("local_name", out var v2) ? v2.GetString() : "meshexec";
                Log("INFO", "advertise_start", new Dictionary<string, object?> { { "service_uuid", svcUuid ?? "" }, { "local_name", localName ?? "" } });
                // Check adapter peripheral support
                try
                {
                    var adapter = await BluetoothAdapter.GetDefaultAsync();
                    if (adapter == null)
                    {
                        throw new Exception("No Bluetooth adapter found");
                    }
                    if (!adapter.IsPeripheralRoleSupported)
                    {
                        throw new Exception("Peripheral role (advertising) not supported on this adapter");
                    }
                    Log("DEBUG", "adapter OK for advertising", new Dictionary<string, object?> { { "BTAddr", adapter.BluetoothAddress } });
                }
                catch (Exception ex)
                {
                    Log("ERROR", "adapter check failed", new Dictionary<string, object?> { { "error", ex.Message } });
                    throw;
                }
                var adv = new BluetoothLEAdvertisement();
                if (!string.IsNullOrWhiteSpace(localName)) adv.LocalName = localName;
                var svcParsed = Guid.TryParse(svcUuid, out var g);
                if (svcParsed)
                {
                    adv.ServiceUuids.Add(g);
                }
                Log("DEBUG", "adv composed", new Dictionary<string, object?> { { "localName", adv.LocalName ?? "" }, { "svcParsed", svcParsed }, { "svcUuidCount", adv.ServiceUuids.Count } });
                publisher?.Stop();
                publisher = new BluetoothLEAdvertisementPublisher(adv);
                publisher.StatusChanged += (s, e) => Log("INFO", "publisher status", new Dictionary<string, object?> { { "status", e.Status.ToString() } });
                try
                {
                    var sw = Stopwatch.StartNew();
                    publisher.Start();
                    sw.Stop();
                    Log("INFO", "advertising started", new Dictionary<string, object?> { { "elapsed_ms", sw.ElapsedMilliseconds } });
                }
                catch (Exception ex)
                {
                    Log("ERROR", "advertising failed", new Dictionary<string, object?> { { "error", ex.Message }, { "stack", ex.ToString() } });
                    // Fallback 1: try minimal advertisement (LocalName only)
                    try
                    {
                        Log("WARN", "retrying with minimal advertisement", null);
                        publisher?.Stop();
                        var adv2 = new BluetoothLEAdvertisement();
                        if (!string.IsNullOrWhiteSpace(localName)) adv2.LocalName = localName;
                        var pub2 = new BluetoothLEAdvertisementPublisher(adv2);
                        pub2.StatusChanged += (s, e) => Log("INFO", "publisher(min) status", new Dictionary<string, object?> { { "status", e.Status.ToString() } });
                        pub2.Start();
                        publisher = pub2;
                        Log("INFO", "advertising started (fallback-minimal)", null);
                    }
                    catch (Exception ex2)
                    {
                        Log("ERROR", "minimal advertising failed", new Dictionary<string, object?> { { "error", ex2.Message }, { "stack", ex2.ToString() } });
                        // Fallback 2: try service UUID only (no local name)
                        try
                        {
                            Log("WARN", "retrying with service-uuid only", null);
                            publisher?.Stop();
                            var adv3 = new BluetoothLEAdvertisement();
                            if (Guid.TryParse(svcUuid, out var g3)) { adv3.ServiceUuids.Add(g3); }
                            var pub3 = new BluetoothLEAdvertisementPublisher(adv3);
                            pub3.StatusChanged += (s, e) => Log("INFO", "publisher(uuid) status", new Dictionary<string, object?> { { "status", e.Status.ToString() } });
                            pub3.Start();
                            publisher = pub3;
                            Log("INFO", "advertising started (fallback-uuid)", null);
                        }
                        catch (Exception ex3)
                        {
                            Log("ERROR", "uuid-only advertising failed", new Dictionary<string, object?> { { "error", ex3.Message }, { "stack", ex3.ToString() } });
                            // Fallback 3: attempt GATT service advertising (WinRT-hosted)
                            try
                            {
                                Log("WARN", "retrying via GATT service advertising", null);
                                if (!Guid.TryParse(svcUuid, out var svcGuid))
                                {
                                    throw new Exception("invalid service UUID for GATT advertising");
                                }
                                var gattRes = await GattServiceProvider.CreateAsync(svcGuid);
                                serviceProvider = gattRes.ServiceProvider;
                                var advParams = new GattServiceProviderAdvertisingParameters
                                {
                                    IsConnectable = true,
                                    IsDiscoverable = true,
                                };
                                serviceProvider.StartAdvertising(advParams);
                                Log("INFO", "GATT service advertising started", null);
                            }
                            catch (Exception ex4)
                            {
                                Log("ERROR", "GATT advertising failed", new Dictionary<string, object?> { { "error", ex4.Message }, { "stack", ex4.ToString() } });
                                throw;
                            }
                        }
                    }
                }
                return JsonDocument.Parse("{\"ok\":true}").RootElement;
            }
        case "advertise_stop":
            {
                publisher?.Stop();
                publisher = null;
                Log("INFO", "advertising stopped", null);
                return JsonDocument.Parse("{\"ok\":true}").RootElement;
            }
        case "gatt_create":
            {
                var p = req.GetProperty("params");
                var svcUuid = p.GetProperty("service_uuid").GetString();
                var chrUuid = p.GetProperty("characteristic_uuid").GetString();
                Log("INFO", "gatt_create", new Dictionary<string, object?> { { "service_uuid", svcUuid ?? "" }, { "characteristic_uuid", chrUuid ?? "" } });
                if (!Guid.TryParse(svcUuid, out var sg) || !Guid.TryParse(chrUuid, out var cg))
                    throw new Exception("invalid UUIDs");

                var res = await GattServiceProvider.CreateAsync(sg);
                serviceProvider = res.ServiceProvider;

                var chrParams = new GattLocalCharacteristicParameters
                {
                    CharacteristicProperties = GattCharacteristicProperties.Read | GattCharacteristicProperties.Write | GattCharacteristicProperties.Notify,
                    ReadProtectionLevel = GattProtectionLevel.Plain,
                    WriteProtectionLevel = GattProtectionLevel.Plain,
                    UserDescription = "MeshExec Characteristic"
                };
                var chrRes = await serviceProvider.Service.CreateCharacteristicAsync(cg, chrParams);
                characteristic = chrRes.Characteristic;
                Log("DEBUG", "gatt characteristic created", new Dictionary<string, object?> { { "svcUuid", svcUuid! }, { "chrUuid", chrUuid! }, { "properties", chrParams.CharacteristicProperties.ToString() } });

                // Broadcast incoming writes to subscribed TCP clients
                characteristic.WriteRequested += async (s, e) =>
                {
                    var deferral = e.GetDeferral();
                    try
                    {
                        var req2 = await e.GetRequestAsync();
                        var len = req2.Value?.Length ?? 0;
                        Log("INFO", "gatt write request", new Dictionary<string, object?> { { "len", len } });
                        if (req2.Value != null)
                        {
                            var bytes = new byte[req2.Value.Length];
                            DataReader.FromBuffer(req2.Value).ReadBytes(bytes);
                            var b64 = Convert.ToBase64String(bytes);
                            var dataObj = new Dictionary<string, object?> { ["event"] = "gatt_write", ["value_b64"] = b64 };
                            var payload = JsonSerializer.Serialize(new { ok = true, data = dataObj });
                            await BroadcastAsync(payload);
                        }
                        req2.Respond();
                    }
                    finally { deferral.Complete(); }
                };

                serviceProvider.AdvertisementStatusChanged += (s, e) => Log("INFO", "gatt adv status", new Dictionary<string, object?> { { "status", e.Status.ToString() } });
                try
                {
                    serviceProvider.StartAdvertising();
                    Log("INFO", "gatt advertising started", null);
                }
                catch (Exception ex)
                {
                    Log("ERROR", "gatt advertising failed", new Dictionary<string, object?> { { "error", ex.Message }, { "stack", ex.ToString() } });
                    throw;
                }
                return JsonDocument.Parse("{\"ok\":true}").RootElement;
            }
        case "gatt_notify":
            {
                if (characteristic == null)
                    throw new Exception("characteristic not created");
                var p = req.GetProperty("params");
                var b64 = p.GetProperty("value_b64").GetString() ?? "";
                var bytes = Convert.FromBase64String(b64);
                Log("DEBUG", "gatt_notify", new Dictionary<string, object?> { { "len", bytes.Length } });
                await characteristic.NotifyValueAsync(bytes.AsBuffer());
                return JsonDocument.Parse("{\"ok\":true}").RootElement;
            }
        case "central_broadcast":
            {
                var p = req.GetProperty("params");
                var svcUuid = p.GetProperty("service_uuid").GetString() ?? string.Empty;
                var chrUuid = p.GetProperty("characteristic_uuid").GetString() ?? string.Empty;
                var b64 = p.GetProperty("value_b64").GetString() ?? string.Empty;
                var scanMs = p.TryGetProperty("scan_ms", out var vScan) ? vScan.GetInt32() : 800;
                if (!Guid.TryParse(svcUuid, out var svc) || !Guid.TryParse(chrUuid, out var chr))
                    throw new Exception("invalid service/characteristic UUIDs");
                var data = Convert.FromBase64String(b64);
                var watcher = new BluetoothLEAdvertisementWatcher
                {
                    ScanningMode = BluetoothLEScanningMode.Active
                };
                int writeCount = 0;
                var attempted = new HashSet<ulong>();
                watcher.Received += async (s, e) =>
                {
                    try
                    {
                        bool hasSvc = e.Advertisement.ServiceUuids.Contains(svc);
                        if (attempted.Contains(e.BluetoothAddress)) return;
                        attempted.Add(e.BluetoothAddress);
                        Log("INFO", hasSvc ? "central match" : "central probe", new Dictionary<string, object?> { { "addr", e.BluetoothAddress }, { "rssi", e.RawSignalStrengthInDBm }, { "hasSvc", hasSvc } });
                        var dev = await BluetoothLEDevice.FromBluetoothAddressAsync(e.BluetoothAddress);
                        if (dev == null) return;
                        var result = await dev.GetGattServicesForUuidAsync(svc);
                        if (result.Status != GattCommunicationStatus.Success) return;
                        foreach (var service in result.Services)
                        {
                            var chrs = await service.GetCharacteristicsForUuidAsync(chr);
                            if (chrs.Status != GattCommunicationStatus.Success) continue;
                            foreach (var c in chrs.Characteristics)
                            {
                                GattCommunicationStatus status;
                                try
                                {
                                    status = await c.WriteValueAsync(data.AsBuffer(), GattWriteOption.WriteWithoutResponse);
                                }
                                catch
                                {
                                    status = await c.WriteValueAsync(data.AsBuffer(), GattWriteOption.WriteWithResponse);
                                }
                                Log("INFO", "central write", new Dictionary<string, object?> { { "status", status.ToString() }, { "len", data.Length }, { "addr", dev.BluetoothAddress } });
                                if (status == GattCommunicationStatus.Success) { writeCount++; }
                            }
                        }
                    }
                    catch (Exception ex)
                    {
                        Log("ERROR", "central write failed", new Dictionary<string, object?> { { "error", ex.Message } });
                    }
                };
                Log("INFO", "central_broadcast start", new Dictionary<string, object?> { { "scan_ms", scanMs } });
                watcher.Start();
                await Task.Delay(scanMs);
                watcher.Stop();
                Log("INFO", "central_broadcast stop", new Dictionary<string, object?> { { "writes", writeCount } });
                return JsonDocument.Parse("{\"ok\":true}").RootElement;
            }
        case "central_write_to":
            {
                var p = req.GetProperty("params");
                var svcUuid = p.GetProperty("service_uuid").GetString() ?? string.Empty;
                var chrUuid = p.GetProperty("characteristic_uuid").GetString() ?? string.Empty;
                var b64 = p.GetProperty("value_b64").GetString() ?? string.Empty;
                if (!Guid.TryParse(svcUuid, out var svc) || !Guid.TryParse(chrUuid, out var chr))
                    throw new Exception("invalid service/characteristic UUIDs");
                var data = Convert.FromBase64String(b64);
                var writes = 0;
                if (p.TryGetProperty("addresses", out var addrs) && addrs.ValueKind == JsonValueKind.Array)
                {
                    foreach (var a in addrs.EnumerateArray())
                    {
                        try
                        {
                            var s = (a.GetString() ?? string.Empty).Replace(":", "").Replace("-", "");
                            if (ulong.TryParse(s, System.Globalization.NumberStyles.HexNumber, null, out var addr))
                            {
                                var dev = await BluetoothLEDevice.FromBluetoothAddressAsync(addr);
                                if (dev == null) continue;
                                var result = await dev.GetGattServicesForUuidAsync(svc);
                                if (result.Status != GattCommunicationStatus.Success) continue;
                                foreach (var service in result.Services)
                                {
                                    var chrs = await service.GetCharacteristicsForUuidAsync(chr);
                                    if (chrs.Status != GattCommunicationStatus.Success) continue;
                                    foreach (var c in chrs.Characteristics)
                                    {
                                        GattCommunicationStatus status;
                                        try { status = await c.WriteValueAsync(data.AsBuffer(), GattWriteOption.WriteWithoutResponse); }
                                        catch { status = await c.WriteValueAsync(data.AsBuffer(), GattWriteOption.WriteWithResponse); }
                                        Log("INFO", "central write(addr)", new Dictionary<string, object?> { { "status", status.ToString() }, { "len", data.Length }, { "addr", addr } });
                                        if (status == GattCommunicationStatus.Success) writes++;
                                    }
                                }
                            }
                        }
                        catch (Exception ex)
                        {
                            Log("ERROR", "central write(addr) failed", new Dictionary<string, object?> { { "error", ex.Message } });
                        }
                    }
                }
                Log("INFO", "central_write_to done", new Dictionary<string, object?> { { "writes", writes } });
                return JsonDocument.Parse("{\"ok\":true}").RootElement;
            }
        // gatt_subscribe / gatt_unsubscribe are handled in ServeAsync where writer is in scope
        default:
            throw new NotSupportedException($"unknown action {action}");
    }
}

int clientCount = 0;

async Task ServeAsync(TcpClient client)
{
    using var c = client;
    using var stream = c.GetStream();
    using var reader = new StreamReader(stream, Encoding.UTF8);
    using var writer = new StreamWriter(stream, new UTF8Encoding(false)) { AutoFlush = true };

    Interlocked.Increment(ref clientCount);
    Log("INFO", "client connected", new Dictionary<string, object?> { { "remote", c.Client.RemoteEndPoint?.ToString() ?? "" }, { "clients", clientCount } });
    while (true)
    {
        var line = await reader.ReadLineAsync();
        if (line == null) break;
        try
        {
            Log("DEBUG", "recv line", new Dictionary<string, object?> { { "len", line.Length }, { "line", line } });
            using var doc = JsonDocument.Parse(line);
            var root = doc.RootElement;
            var action = root.GetProperty("action").GetString() ?? string.Empty;
            if (action == "gatt_subscribe")
            {
                lock (subLock) { subscribers.Add(writer); }
                var ok = JsonDocument.Parse("{\"ok\":true}").RootElement.GetRawText();
                await writer.WriteLineAsync(ok);
                continue;
            }
            if (action == "gatt_unsubscribe")
            {
                lock (subLock) { subscribers.Remove(writer); }
                var ok = JsonDocument.Parse("{\"ok\":true}").RootElement.GetRawText();
                await writer.WriteLineAsync(ok);
                continue;
            }
            var resp = await HandleAsync(root);
            var outLine = resp.GetRawText();
            Log("DEBUG", "send line", new Dictionary<string, object?> { { "len", outLine.Length }, { "line", outLine } });
            await writer.WriteLineAsync(outLine);
        }
        catch (Exception ex)
        {
            var err = JsonSerializer.Serialize(new { ok = false, error = ex.Message });
            await writer.WriteLineAsync(err);
            Log("ERROR", "request error", new Dictionary<string, object> { { "error", ex.Message }, { "stack", ex.ToString() } });
        }
    }
    Interlocked.Decrement(ref clientCount);
    Log("INFO", "client disconnected", new Dictionary<string, object?> { { "clients", clientCount } });
}

// Periodic health log
_ = Task.Run(async () =>
{
    while (true)
    {
        try
        {
            Log("DEBUG", "health", new Dictionary<string, object> { { "mem_bytes", GC.GetTotalMemory(false) }, { "clients", clientCount }, { "publisherActive", publisher != null }, { "serviceProviderActive", serviceProvider != null } });
        }
        catch { }
        await Task.Delay(30000);
    }
});

while (true)
{
    var client = await server.AcceptTcpClientAsync();
    _ = ServeAsync(client);
}



