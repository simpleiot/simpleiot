module Api.Response exposing (Response, decoder)

import Json.Decode as Decode
import Json.Decode.Pipeline exposing (optional, required)


type alias Response =
    { success : Bool
    , error : String
    , id : String
    }


decoder : Decode.Decoder Response
decoder =
    Decode.succeed Response
        |> required "success" Decode.bool
        |> optional "error" Decode.string ""
        |> optional "id" Decode.string ""
