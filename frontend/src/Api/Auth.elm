module Api.Auth exposing
    ( Cred
    , Response
    , login
    )

import Api.Data exposing (Data)
import Http
import Json.Decode as Decode
import Json.Decode.Pipeline exposing (required)
import Url.Builder


type alias Cred =
    { email : String
    , password : String
    }


type alias Response =
    { token : String
    , isRoot : Bool
    }


decodeResponse : Decode.Decoder Response
decodeResponse =
    Decode.succeed Response
        |> required "token" Decode.string
        |> required "isRoot" Decode.bool


login : Cred -> (Data Response -> msg) -> Cmd msg
login cred onResponse =
    Http.post
        { body =
            Http.multipartBody
                [ Http.stringPart "email" cred.email
                , Http.stringPart "password" cred.password
                ]
        , url = Url.Builder.absolute [ "v1", "auth" ] []
        , expect = Api.Data.expectJson onResponse decodeResponse
        }
