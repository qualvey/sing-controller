import { getGeoIPInfoSync, type IPInfo } from '@/api/geoip'
import { ref } from 'vue'

const enabled = ref(false)

const EMPTY_GEOIP_INFO: IPInfo = {
  ip: '',
  country: '',
  region: '',
  city: '',
  asn: '',
  organization: '',
}

export const setConnectionGeoIPEnabled = (value: boolean) => {
  enabled.value = value
}

export const getConnectionGeoIPInfoSync = (ip: string): IPInfo => {
  if (!enabled.value) {
    return EMPTY_GEOIP_INFO
  }

  return getGeoIPInfoSync(ip)
}
