use serde::de::Error as DeError;
use serde::{Deserialize, Deserializer, Serializer};
use time::{format_description::well_known::Rfc3339, OffsetDateTime};

pub fn serialize_datetime<S>(datetime: &OffsetDateTime, serializer: S) -> Result<S::Ok, S::Error>
where
    S: Serializer,
{
    let iso_string = datetime.format(&Rfc3339).unwrap();
    serializer.serialize_str(&iso_string)
}

pub fn deserialize_datetime<'de, D>(deserializer: D) -> Result<OffsetDateTime, D::Error>
where
    D: Deserializer<'de>,
{
    let iso_string = String::deserialize(deserializer)?;
    OffsetDateTime::parse(&iso_string, &Rfc3339).map_err(DeError::custom)
}

pub fn serialize_option_datetime<S>(
    datetime: &Option<OffsetDateTime>,
    serializer: S,
) -> Result<S::Ok, S::Error>
where
    S: Serializer,
{
    match datetime {
        Some(dt) => serialize_datetime(dt, serializer),
        None => serializer.serialize_none(),
    }
}

pub fn deserialize_option_datetime<'de, D>(
    deserializer: D,
) -> Result<Option<OffsetDateTime>, D::Error>
where
    D: Deserializer<'de>,
{
    let iso_string = Option::<String>::deserialize(deserializer)?;
    match iso_string {
        Some(s) => OffsetDateTime::parse(&s, &Rfc3339)
            .map(Some)
            .map_err(DeError::custom),
        None => Ok(None),
    }
}
