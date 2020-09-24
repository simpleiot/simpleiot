module Api.Data exposing
    ( Data(..)
    , errorToString
    , expectJson
    , map
    , toMaybe
    )

import Http
import Json.Decode as Json


type Data value
    = NotAsked
    | Loading
    | Failure Http.Error
    | Success value


map : (a -> b) -> Data a -> Data b
map fn data =
    case data of
        NotAsked ->
            NotAsked

        Loading ->
            Loading

        Failure reason ->
            Failure reason

        Success value ->
            Success (fn value)


toMaybe : Data value -> Maybe value
toMaybe data =
    case data of
        Success value ->
            Just value

        _ ->
            Nothing


expectJson : (Data value -> msg) -> Json.Decoder value -> Http.Expect msg
expectJson toMsg decoder =
    Http.expectStringResponse (fromResult >> toMsg) <|
        \response ->
            case response of
                Http.BadUrl_ url ->
                    Err (Http.BadUrl url)

                Http.Timeout_ ->
                    Err Http.Timeout

                Http.NetworkError_ ->
                    Err Http.NetworkError

                Http.BadStatus_ metadata _ ->
                    Err (Http.BadStatus metadata.statusCode)

                Http.GoodStatus_ _ body ->
                    case Json.decodeString decoder body of
                        Ok value ->
                            Ok value

                        Err err ->
                            Err (Http.BadBody (Json.errorToString err))


fromResult : Result Http.Error value -> Data value
fromResult result =
    case result of
        Ok value ->
            Success value

        Err reasons ->
            Failure reasons


errorToString : Http.Error -> String
errorToString err =
    case err of
        Http.BadUrl url ->
            "Malformed url: " ++ url

        Http.Timeout ->
            "Timeout exceeded"

        Http.NetworkError ->
            "Network error"

        Http.BadStatus resp ->
            "Bad status: " ++ String.fromInt resp

        Http.BadBody resp ->
            "Bad body: " ++ resp
