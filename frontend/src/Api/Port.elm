port module Api.Port exposing
    ( Clipboard
    , encodeClipboard
    , out
    )

import Json.Encode


port out : Json.Encode.Value -> Cmd msg


type alias Clipboard =
    { action : String
    , data : String
    }


encodeClipboard : String -> Json.Encode.Value
encodeClipboard s =
    Json.Encode.object
        [ ( "action", Json.Encode.string "CLIPBOARD" )
        , ( "data", Json.Encode.string s )
        ]
