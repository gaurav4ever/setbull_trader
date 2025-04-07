import time
import json
import requests
from google.protobuf.json_format import MessageToDict
import grpc
from your_proto_module import BizdirectIamMessages_pb2, BizdirectIamService_pb2_grpc  # Adjust import paths accordingly
from crypt_util import encrypt  # You must implement this to match Java's CryptUtil.encrypt

def get_mobile_numbers():
    return [
        "8555068264", "9455590235", "9955535995",
        "9355583064", "9255539885", "9255573252",
        "8555806949", "9955526060", "8555834675"
    ]

def get_metadata():
    metadata = [
        ('x-device-identifier', '1'),
        ('x-device-type', '1'),
        ('x-mobile-version', '1'),
        ('x-os-version', '1'),
        ('x-app-version', '1.0.63672763'),
        ('x-app-version-code', '100'),
    ]
    return metadata

def generate_auth_token(mobile):
    deviceid = "deviceid-303"
    current_time = str(int(time.time() * 1000))
    countrycode = "91"

    payload = f"{mobile}|{deviceid}|{current_time}|{countrycode}"
    partner_secret_key = "7MERQhZjF4kGbnvXQrHqcDMWu53WbceF"
    sdk_partner_key = "abcd-1234-efg"

    request_payload = encrypt(payload, partner_secret_key)
    print(request_payload)

    url = "https://callback-preprod.gonuclei.com/api/partner-identity/seamless/v1/generatetoken"
    headers = {"Content-Type": "application/json"}
    data = {
        "payload": request_payload,
        "partner_key": sdk_partner_key
    }

    response = requests.post(url, headers=headers, json=data)
    response_json = response.json()

    encrypted_token = response_json["token"]

    request_proto = BizdirectIamMessages_pb2.SeamlessRequest(
        partner_key=sdk_partner_key,
        device_id=deviceid,
        mobile=mobile,
        country_code=91,
        encrypted_temp_token=encrypted_token
    )

    # gRPC connection
    channel = grpc.secure_channel("preprod-az.gonuclei.com:443", grpc.ssl_channel_credentials())
    stub = BizdirectIamService_pb2_grpc.BizdirectIamServiceStub(channel)

    response_proto = stub.seamlessLogin(request_proto, metadata=get_metadata())
    auth_token = response_proto.auth_token
    print("Bearer", auth_token)
    return f"Bearer {auth_token}"

def main():
    mobs = get_mobile_numbers()
    tokens = []

    for mob in mobs:
        token = generate_auth_token(mob)
        tokens.append(token)

    print("----------------------------------------")
    for t in tokens:
        print(t)

if __name__ == "__main__":
    main()
