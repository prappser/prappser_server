from jwcrypto import jwe, jwk
import base64
import hashlib

# The password you used to encrypt
password = "Prappser2025"

# Step 1: derive key (MD5 digest of password)
key_bytes = hashlib.md5(password.encode('utf-8')).digest()  # 16 bytes

# Step 2: Base64url encode without padding, for jwcrypto's JWK
key_b64url = base64.urlsafe_b64encode(key_bytes).rstrip(b'=').decode('utf-8')

# Step 3: create the symmetric key JWK
key = jwk.JWK(kty='oct', k=key_b64url)

# Step 4: your JWE token (paste your full token here)
token = "eyJlbmMiOiJBMTI4R0NNIiwiYWxnIjoiZGlyIn0..914U_1q7qRgmah7-.4MqUDKtqo1CH2Y2JcFdsrelwHBBFOK0FJKXGqjwKCXo1XVgm0CuHDh6sBaDKDuDjtYtNW8Mrx_n7NPhWpchbL-ejwI8LWcS4VTBM0sVqMxgzNTckW2vbEXKTnTQlpF_-MIfLFANr5EskLFuUIpeZTBN_KZ5bqeMn5Uwjth0CXrQ6ec_2nSRnK7yMIShCRTjSiOBUhrT-d0tHQjbKXMpqyQ4GQYVwQ5g-wRT7vsOpBBZmw5YpuSeL32ZD7Ltl_bnKAWkis55Uj2nu6oCZN-chqMDDetHjuHYGRSf4GbrGB2mJ_T7EzUjyHVrtAUS6ZYaLTglcrwH1TZ3dIv0oeexDECfrJjvAP_VcNXmgP5OB6_kS937wquuoNgnjTNnMvL9QGNfIbUW4tL7QcGyjJ4ynQcheWAIUPu9Y73Se-9ecDnAr_Tq83EKWUFFyhQb_lTxULtRQ5GqrC-vYeIXy63BsJqSUwr1hPNXP6Pm3_IJzwK4HtjyZWwQzxH3gBogDhP39MB5eRCkUPHaMyLyGZ1PYEUlrXabZp0BsxIDqK6sWlNj0JP63diHb8INLR7ysGD2SMOzsuxfhJtzCnK2SHK5hNOSXpcDYd1mWcKz1lh_iCPk--AYqGX27r2Doa06jFDqeMt31zrulDbwwQgW-_wKtY6VRk-Yb-M8_Dpy-qZFt1GzTObYCa9ovIFqDbt5hUPa_e2EYBesGoDHG4GXz_e4d4ACObJSSxCBt0s9ywgviw-7Oc6aRy6z8u_bqrY315kbQcRI4mFLgWPHgHDZOtEbJDxk-uXdg0DlrecMh2VP1Gfn3JUVCy3TUlc_f-grnQfUWgMrcI0c1jx49rzoVBx7sg0o1jzwQyozYRwKWekS2r-Xi3z1218yfwFNYImaaJgvjbooyvb_gMD3H2Gd6nktLS9hEpaUvJ4rhNiFNgsYuBSg4Pv6u9DPsZyeUDUcUBhX_oh931IVaZj81ZPkfSLuiHWwpNAkhA6KAqn0xl8MSBOMakcY6hZfDSGQwdBBIaCGq_ryRRTDP26Y8LeOt0Kny5qP-b7RcHsiR0gfvrrfqWBE9u9jiMvR5mOnGUKHegSjQpyrSbQO-y-GrX5a_r4_fPWNzyWHXhq88-KfqEegdSxEU4TR4ji2hD1ofhvCQFXstjT6B96YyLxN9855yUzZCjCSFbJ2MVpAnIoVlTiLtLBRTSqkfOI58S5XLw_Bgx98C6cD13XuTs30VnKf28_bDxkyuGqezrTVBjRSSjmKma8KzOpD_97ZO9EZkr2AT5MerN4-_gCbhxthL19HB-RnPHhMHf04KTLmfqojfLeljOobSRMpPkA.X-BZvdm7Nw48ckMrX2cWnA"
# Step 5: decrypt
jwetoken = jwe.JWE()
jwetoken.deserialize(token)
jwetoken.decrypt(key)

# Step 6: print payload (should be JSON containing 'jws' field)
print(jwetoken.payload.decode('utf-8'))
