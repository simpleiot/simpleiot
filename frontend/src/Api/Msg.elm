module Api.Msg exposing (send)

import Api.Data exposing (Data)
import Api.Point as Point
import Api.Response as Response exposing (Response)
import Http
import Time
import Url.Builder


send :
    { token : String
    , msg : String
    , now : Time.Posix
    , onResponse : Data Response -> msg
    }
    -> Cmd msg
send options =
    let
        empty =
            Point.empty

        point =
            { empty | text = options.msg, time = options.now }
    in
    Http.request
        { method = "POST"
        , headers = [ Http.header "Authorization" <| "Bearer " ++ options.token ]
        , url = Url.Builder.absolute [ "v1", "msg" ] []
        , expect = Api.Data.expectJson options.onResponse Response.decoder
        , body = point |> Point.encode |> Http.jsonBody
        , timeout = Nothing
        , tracker = Nothing
        }
